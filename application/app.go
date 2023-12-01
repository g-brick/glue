package application

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/glue/clients/mysql"

	"github.com/glue/logger"
	"github.com/glue/monitor"
	"github.com/glue/services/conf"
	"github.com/glue/services/consumer"
	"github.com/glue/services/grpc"
	"github.com/glue/services/http"
	"github.com/glue/services/producer"
	"github.com/glue/services/tcp"

	"github.com/onsi/ginkgo/config"
	"go.uber.org/zap/zapcore"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	BuildTime string
	GitCommit string
	GoVersion string
	Version   bool
)

func init() {
	flag.BoolVar(&Version, "v", false, "show version")
}

// Service represents a service attached to the server.
type Service interface {
	WithLogger(log *zap.Logger)
	Init()
	Start() error
	Stop() error
}

func Default() *App {
	app := &App{}
	app.Init()
	return app
}

type App struct {
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	Logger *zap.Logger
	conf.Name

	GRPCService *grpc.Service
	HTTPService *http.Service
	TCPService  *tcp.Service

	ConfigService   *conf.Service
	ConsumerService *consumer.Service
	MonitorService  *monitor.Service
	MySQLClient     *mysql.Client

	ProducerService *producer.Service

	Configurator

	OtherServices  map[string]Service
	initedServices map[string]Service

	Reporters map[string]monitor.Reporter

	isInit bool

	debug bool

	logLevel zapcore.Level
}

func (app *App) Init() {
	app.initApp()
	app.UnmarshalConfigurator(NewConfig())
	app.initCommon()
}

func (app *App) InitWithConfig(cfg Configurator) {
	app.initApp()
	app.UnmarshalConfigurator(cfg)
	app.initCommon()
}

func (app *App) InitWithConfigAndOptions(cfg Configurator, options []interface{}) {
	app.initApp()
	app.UnmarshalConfigurator(cfg)
	app.initCommonWithOptions(options)
}

func (app *App) PrintConfig(cfg interface{}) {
	content, err := json.Marshal(cfg)
	fmt.Println("config=", string(content), "err=", err)
}

func (app *App) initApp() {
	if !flag.Parsed() {
		flag.Parse()
	}

	if Version {
		fmt.Printf("version: %s\ngo version: %s\ngit commit: %s\nbuild time: %s\n", config.VERSION, GoVersion, GitCommit, BuildTime)
		os.Exit(0)
	}

	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.initLogger(conf.DefaultLogLevel)

	app.Logger.Info("Init App.")

	app.initName()
	app.initConfigService()
	app.updateLogger()
}

func (app *App) initCommon() {
	rand.Seed(time.Now().UnixNano())

	app.initGRPCService(nil)
	app.initHTTPService()
	app.initTCPService()
	app.initConsumerService()
	app.initMySQLClient()
	app.initProducerService()
	app.initMonitorService()
	app.isInit = true
}

func (app *App) initCommonWithOptions(options []interface{}) {
	rand.Seed(time.Now().UnixNano())

	// 需要定制options的服务，陆续在这里添加
	app.initGRPCService(options)
	app.initHTTPService()
	app.initTCPService()
	app.initConsumerService()
	app.initMySQLClient()
	app.initProducerService()
	app.initMonitorService()
	app.isInit = true
}

func (app *App) initLogger(level string) {
	cfg := logger.NewConfig()

	n := zap.NewAtomicLevel()
	if err := n.UnmarshalText([]byte(level)); err == nil {
		cfg.Level = n.Level()
	}

	app.logLevel = cfg.Level

	l, _ := cfg.New(os.Stderr)

	//TODO: service 变量最好通过build的时候打入到main
	l.With(
		zap.String("Service", app.Service),
		zap.String("Cluster", app.Cluster),
		zap.String("Instance", app.Instance),
	)
	app.Logger = l
	app.updateServiceLogger()
}

func (app *App) updateLogger() {
	app.Logger.Info("Update Logger.")
	logLevel, err := app.ConfigService.Get(conf.LogLevelConfigName)
	if err != nil {
		app.Logger.Panic("failed to get log level from config service.",
			zap.Error(err))
	}

	if logLevel != nil {
		level, ok := logLevel.(string)
		if ok {
			app.initLogger(level)
		}
	} else {
		app.initLogger(conf.DefaultLogLevel)
	}
	app.Logger.Debug("Current Logger Level", zap.Any("loglevel", logLevel))
}

func (app *App) updateServiceLogger() {
	if app.ConfigService != nil {
		app.ConfigService.WithLogger(app.Logger)
	}
	if app.MonitorService != nil {
		app.MonitorService.WithLogger(app.Logger)
	}
	if app.ProducerService != nil {
		app.ProducerService.WithLogger(app.Logger)
	}
	if app.ConsumerService != nil {
		app.ConsumerService.WithLogger(app.Logger)
	}
	if app.HTTPService != nil {
		app.HTTPService.WithLogger(app.Logger)
	}
	if app.GRPCService != nil {
		app.GRPCService.WithLogger(app.Logger)
	}
	if app.TCPService != nil {
		app.TCPService.WithLogger(app.Logger)
	}
}

func (app *App) Context() context.Context {
	if app.ctx == nil {
		app.ctx, app.cancel = context.WithCancel(context.Background())
	}
	return app.ctx
}

func (app *App) initName() {
	hostname, _ := os.Hostname()
	name := conf.Name{
		Instance: hostname,
		Cluster:  hostname,
		Service:  filepath.Base(os.Args[0]),
	}
	app.Name = name
}

func (app *App) initMonitorService() {
	app.MonitorService = monitor.NewService(app.ctx, app, app.Configurator.Configuration().Monitor)
	app.MonitorService.WithLogger(app.Logger)
	app.MonitorService.Init()

	app.MonitorService.SetGlobalTag("module", app.Configurator.Configuration().Global.Module)
	app.MonitorService.SetGlobalTag("group", app.Configurator.Configuration().Global.Group)

	app.HTTPService.Monitor = app.MonitorService
}

func (app *App) RegisterNewReporter(name string, report monitor.Reporter) {
	if app.Reporters == nil {
		app.Reporters = make(map[string]monitor.Reporter)
	}

	app.Reporters[name] = report
}

// Statistics returns statistics for the services running in the Server.
func (app *App) Statistics(tags map[string]string) []monitor.Statistic {
	var statistics []monitor.Statistic
	statistics = append(statistics, app.ConsumerService.Statistics(tags)...)
	statistics = append(statistics, app.ProducerService.Statistics(tags)...)
	for _, srv := range app.OtherServices {
		if m, ok := srv.(monitor.Reporter); ok {
			statistics = append(statistics, m.Statistics(tags)...)
		}
	}

	for _, report := range app.Reporters {
		statistics = append(statistics, report.Statistics(tags)...)
	}
	return statistics
}

func (app *App) initOtherServices() {
	for srvName, srv := range app.OtherServices {
		if _, ok := app.initedServices[srvName]; !ok {
			app.Logger.Info("Init New Service.", zap.String("Service", srvName))
			srv.WithLogger(app.Logger)
			srv.Init()
		}
		app.initedServices[srvName] = srv
	}
}

func (app *App) startOtherServices() error {
	for _, srv := range app.OtherServices {
		if err := srv.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (app *App) stopOtherServices() error {
	for _, srv := range app.OtherServices {
		if err := srv.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (app *App) GetServiceByName(name string) (Service, bool) {
	if app.OtherServices == nil {
		app.OtherServices = make(map[string]Service)
	}
	app.mu.RLock()
	src, ok := app.OtherServices[name]
	app.mu.RUnlock()
	return src, ok
}

func (app *App) RegisterNewService(name string, srv Service) {
	app.Logger.Info("Register New Service.", zap.String("Service", name))
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.OtherServices == nil {
		app.OtherServices = make(map[string]Service)
	}

	if app.initedServices == nil {
		app.initedServices = make(map[string]Service)
	}
	app.OtherServices[name] = srv
	app.initOtherServices()
}

func (app *App) initConfigService() {
	app.Logger.Info("Init Config Service")
	config, err := conf.NewConfig()
	if err != nil {
		app.Logger.Panic("failed to create config.",
			zap.Error(err))
	}
	app.ConfigService = conf.NewService(config, app.ctx)
	app.ConfigService.WithLogger(app.Logger)
	if err := app.ConfigService.Init(); err != nil {
		app.Logger.Panic("failed to init Config Service.",
			zap.Error(err))
	}
}

func (app *App) UnmarshalConfigurator(cfg Configurator) {
	app.Logger.Info("Init Global Config.")
	cfg.Configuration().Init()
	cfg.Configuration().cfg = app.ConfigService
	cfg.Configuration().Global = NewGlobalConfig(app.Service)

	if err := cfg.Configuration().Unmarshal(cfg); err != nil {
		app.Logger.Panic("failed to UnmarshalConfigurator.", zap.Error(err))
	}
	app.Configurator = cfg

	content, err := json.Marshal(cfg)
	if err != nil {
		app.Logger.Info("Current Config", zap.ByteString("Config", content))
	}
}

func (app *App) initGRPCService(options []interface{}) {
	app.GRPCService = grpc.NewService(app.ctx, app.Configurator.Configuration().GRPC, options)
	app.GRPCService.WithLogger(app.Logger)
	app.GRPCService.Init()
}

func (app *App) initTCPService() {
	app.TCPService = tcp.NewService(app.ctx, app.Configurator.Configuration().TCP)
	app.TCPService.WithLogger(app.Logger)
	app.TCPService.Init()
}

func (app *App) initHTTPService() {
	app.HTTPService = http.NewService(app.ctx, app.Configurator.Configuration().HTTP)
	app.HTTPService.WithLogger(app.Logger)
	app.HTTPService.Init()
	var res map[string]string
	router := app.HTTPService
	router.GET("/api", func(c *gin.Context) {
		res = make(map[string]string)
		res["/api/config"] = "查看配置文件"
		res["/api/stats"] = "查看统计信息"
		res["/debug/pprof"] = "pprof性能检测"
		c.JSON(200, res)
	})

	router.GET("/api/config", func(c *gin.Context) {
		c.JSON(200, app.Configurator)

	})

	router.GET("/api/debug", func(c *gin.Context) {
		if !app.debug {
			app.debug = true
			app.initLogger("debug")
		} else {
			app.debug = false
			app.updateLogger()
		}
		c.JSON(200, app.logLevel)
	})
}

func (app *App) initConsumerService() {
	app.ConsumerService = consumer.NewService(app.ctx, app.Configurator.Configuration().Consumer)
	app.ConsumerService.WithLogger(app.Logger)
	app.ConsumerService.Init()
}

func (app *App) initMySQLClient() {
	app.MySQLClient = mysql.NewClient(app.Configurator.Configuration().MySQL)
}

func (app *App) initProducerService() {
	app.ProducerService = producer.NewService(app.ctx, app.Configurator.Configuration().Producer)
	app.ProducerService.WithLogger(app.Logger)
	app.ProducerService.Init()
}

func (app *App) Start() error {
	app.Logger.Info("Start App")

	if !app.isInit {
		return fmt.Errorf("app is not init.")
	}

	if err := app.ConfigService.Start(); err != nil {
		return err
	}

	if err := app.GRPCService.Start(); err != nil {
		return err
	}

	if err := app.HTTPService.Start(); err != nil {
		return err
	}
	if err := app.TCPService.Start(); err != nil {
		return err
	}

	if err := app.ConsumerService.Start(); err != nil {
		return err
	}

	if err := app.ProducerService.Start(); err != nil {
		return err
	}

	if err := app.startOtherServices(); err != nil {
		return err
	}

	app.Logger.Info("Wait Signal.....")

	//Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit

	app.Logger.Info("Got Signal. Exit.")

	if err := app.Stop(); err != nil {
		return err
	}

	return nil
}

func (app *App) Stop() error {
	app.Logger.Info("Stop App.")
	if err := app.ConfigService.Stop(); err != nil {
		return err
	}

	if err := app.GRPCService.Stop(); err != nil {
		return err
	}

	if err := app.HTTPService.Stop(); err != nil {
		return err
	}

	if err := app.TCPService.Stop(); err != nil {
		return err
	}

	if err := app.ConsumerService.Stop(); err != nil {
		return err
	}

	if err := app.ProducerService.Stop(); err != nil {
		return err
	}

	if err := app.stopOtherServices(); err != nil {
		return err
	}
	app.cancel()

	app.Logger.Info("Wait 5 Seconds ...")
	time.Sleep(5 * time.Second)
	app.Logger.Info("Server Exited.")
	return nil
}

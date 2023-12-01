package conf

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	DefaultPublicNamespaceEnvName = "PublicNamespace"

	DefaultPublicNamespace = "umonitor3.public"
	DefaultPrimaryNamespce = "application"
	DefaultLogLevel = "info"
	LogLevelConfigName = "log.level"
	DefaultConfPath = "app.properties"
	FileConfigFormatJson = "json"
	FileConfigFormatToml = "toml"
	DefaultFileConfigFormat = FileConfigFormatJson
)

var (
	confPath 			string
	fileConfigFormat 	string
)

func init() {
	flag.StringVar(&confPath, "c", DefaultConfPath, "config file path， default using apollo-confcenter by app.properties")
}

type Name struct {
	Service  string
	Cluster  string
	Instance string
}

type Config struct {
	PublicNamespace 	string `json:"namespace.public"`
	PrimayNamespace 	string `json:"namespace.primary"`
	LogLevel 			string `json:"log.level"`
	ConfPath 			string
	FileConfigFormat	string

}

func NewConfig() (*Config, error) {
	c := &Config{
		PublicNamespace: DefaultPublicNamespace,
		PrimayNamespace: DefaultPrimaryNamespce,
		LogLevel: DefaultLogLevel,
	}

	publicNamespace := os.Getenv(DefaultPublicNamespaceEnvName)
	if len(publicNamespace) > 0 {
		c.PublicNamespace = publicNamespace
	}

	c.ConfPath = confPath

	confPathList := strings.Split(confPath, ".")

	if len(confPathList) != 2 {
		return nil, fmt.Errorf("Invaild Config File Name.")
	}

	if confPathList[1] == FileConfigFormatJson {
		c.FileConfigFormat = FileConfigFormatJson
	} else if confPathList[1] == FileConfigFormatToml {
		c.FileConfigFormat = FileConfigFormatToml
	} else if confPath != DefaultConfPath {
		return nil, fmt.Errorf("Invaild Config File Format. Only suport *.json/*.toml.")
	}

	return c, nil
}
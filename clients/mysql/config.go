package mysql

const (
	DefaultMySQLEndpoint  = "localhost:3306"
	DefaultMySQLUsername  = "root"
	DefaultMySQLPassowrd  = ""
)

type Config struct {
	MySQLEndpoint   	string 	 	`json:"mysql.endpoint" toml:"endpoint"`
	MySQLUsername		string 		`json:"mysql.username" toml:"username"`
	MySQLPassword		string 		`json:"mysql.password" toml:"password"`
}


func NewConfig() Config {
	c := Config{
		MySQLEndpoint: 			DefaultMySQLEndpoint,
		MySQLUsername:	 		DefaultMySQLUsername,
		MySQLPassword: 			DefaultMySQLPassowrd,
	}
	return c
}
package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	ServerTimeOut int `json:"server_time_out"`

	JwtSecretKey string `json:"jwt_secret_key"`

	AdminEmail string `json:"admin_email"`
	AdminID    int64  `json:"admin_id"`
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type PsqlDB struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	SSLMode   string `json:"ssl_mode"`
	DBName    string `json:"db_name"`
	DBMaxOpen int    `json:"db_max_open"`
	DBMaxIdle int    `json:"db_max_idle"`
}

type RabbitmqConfig struct {
	Host     string `json:"host" mapstructure:"RABBITMQ_HOST"`
	Port     string `json:"port" mapstructure:"RABBITMQ_PORT"`
	Username string `json:"username" mapstructure:"RABBITMQ_USER"`
	Password string `json:"password" mapstructure:"RABBITMQ_PASSWORD"`
}

type Email struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      bool   `json:"tls"`
}

type ExchangeName struct {
	OrderEvent string `json:"order_event"`
}

type Config struct {
	App          App            `json:"app"`
	Psql         PsqlDB         `json:"psql"`
	RabbitMQ     RabbitmqConfig `json:"rabbitmq"`
	Redis        RedisConfig    `json:"redis"`
	Email        Email          `json:"email"`
	ExchangeName ExchangeName   `json:"exchange_name"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort:       viper.GetString("APP_PORT"),
			AppEnv:        viper.GetString("APP_ENV"),
			ServerTimeOut: viper.GetInt("SERVER_TIMEOUT"),
			JwtSecretKey:  viper.GetString("JWT_SECRET_KEY"),
			AdminEmail:    viper.GetString("ADMIN_EMAIL"),
			AdminID:       viper.GetInt64("ADMIN_ID"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Psql: PsqlDB{
			Host:      viper.GetString("DATABASE_HOST"),
			Port:      viper.GetString("DATABASE_PORT"),
			User:      viper.GetString("DATABASE_USER"),
			Password:  viper.GetString("DATABASE_PASSWORD"),
			SSLMode:   viper.GetString("DATABASE_SSLMODE"),
			DBName:    viper.GetString("DATABASE_NAME"),
			DBMaxOpen: viper.GetInt("DATABASE_MAX_OPEN_CONNECTION"),
			DBMaxIdle: viper.GetInt("DATABASE_MAX_IDLE_CONNECTION"),
		},
		RabbitMQ: RabbitmqConfig{
			Host:     viper.GetString("RABBITMQ_HOST"),
			Port:     viper.GetString("RABBITMQ_PORT"),
			Username: viper.GetString("RABBITMQ_USERNAME"),
			Password: viper.GetString("RABBITMQ_PASSWORD"),
		},
		Email: Email{
			Host:     viper.GetString("EMAIL_HOST"),
			Port:     viper.GetInt("EMAIL_PORT"),
			Username: viper.GetString("EMAIL_USERNAME"),
			Password: viper.GetString("EMAIL_PASSWORD"),
			From:     viper.GetString("EMAIL_FROM"),
			TLS:      viper.GetBool("EMAIL_TLS"),
		},
		ExchangeName: ExchangeName{
			OrderEvent: viper.GetString("ORDER_EVENT"),
		},
	}
}

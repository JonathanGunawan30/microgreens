package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`

	ServerTimeOut     int    `json:"server_time_out"`
	ProductServiceUrl string `json:"product_service_url"`
	UserServiceUrl    string `json:"user_service_url"`
	OrderServiceUrl   string `json:"order_service_url"`

	InternalKey string `json:"internal_key"`
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

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}
type RabbitmqConfig struct {
	Host     string `json:"host" mapstructure:"RABBITMQ_HOST"`
	Port     string `json:"port" mapstructure:"RABBITMQ_PORT"`
	Username string `json:"username" mapstructure:"RABBITMQ_USER"`
	Password string `json:"password" mapstructure:"RABBITMQ_PASSWORD"`
}

type Midtrans struct {
	ServerKey   string `json:"server_key"`
	Environment int    `json:"environment"`
}

type PublisherName struct {
	PaymentSuccess string `json:"payment_success"`
}

type ExchangeName struct {
	PaymentEvent string `json:"payment_event"`
	OrderEvent   string `json:"order_event"`
	UserEvent    string `json:"user_event"`
}

type QueueName struct {
	OrderSnapshot string `json:"order_snapshot"`
	UserSnapshot  string `json:"user_snapshot"`
}

type Config struct {
	App           App            `json:"app"`
	Psql          PsqlDB         `json:"psql"`
	Redis         RedisConfig    `json:"redis"`
	RabbitMQ      RabbitmqConfig `json:"rabbitmq"`
	Midtrans      Midtrans       `json:"midtrans"`
	PublisherName PublisherName  `json:"publisher_name"`
	ExchangeName  ExchangeName   `json:"exchange_name"`
	QueueName     QueueName      `json:"queue_name"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort:           viper.GetString("APP_PORT"),
			AppEnv:            viper.GetString("APP_ENV"),
			JwtSecretKey:      viper.GetString("JWT_SECRET_KEY"),
			ServerTimeOut:     viper.GetInt("SERVER_TIMEOUT"),
			ProductServiceUrl: viper.GetString("PRODUCT_SERVICE_URL"),
			UserServiceUrl:    viper.GetString("USER_SERVICE_URL"),
			OrderServiceUrl:   viper.GetString("ORDER_SERVICE_URL"),
			InternalKey:       viper.GetString("INTERNAL_KEY"),
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
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		RabbitMQ: RabbitmqConfig{
			Host:     viper.GetString("RABBITMQ_HOST"),
			Port:     viper.GetString("RABBITMQ_PORT"),
			Username: viper.GetString("RABBITMQ_USERNAME"),
			Password: viper.GetString("RABBITMQ_PASSWORD"),
		},
		Midtrans: Midtrans{
			ServerKey:   viper.GetString("MIDTRANS_SERVER_KEY"),
			Environment: viper.GetInt("MIDTRANS_ENVIRONMENT"),
		},
		PublisherName: PublisherName{
			PaymentSuccess: viper.GetString("PUBLISHER_PAYMENT_SUCCESS"),
		},
		ExchangeName: ExchangeName{
			PaymentEvent: viper.GetString("EXCHANGE_PAYMENT_EVENT"),
			OrderEvent:   viper.GetString("EXCHANGE_ORDER_EVENT"),
			UserEvent:    viper.GetString("EXCHANGE_USER_EVENT"),
		},
		QueueName: QueueName{
			OrderSnapshot: viper.GetString("QUEUE_ORDER_SNAPSHOT_DB"),
			UserSnapshot:  viper.GetString("QUEUE_USER_SNAPSHOT_DB"),
		},
	}
}

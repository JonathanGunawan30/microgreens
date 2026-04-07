package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`

	ServerTimeOut     int    `json:"server_time_out"`
	ProductServiceUrl string `json:"product_service_url"`
	UserServiceUrl    string `json:"user_service_url"`

	LatitudeRef  string `json:"latitude_ref"`
	LongitudeRef string `json:"longitude_ref"`
	MaxDistance  int    `json:"max_distance"`

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

type PublisherName struct {
	ProductUpdateStock    string `json:"product_update_stock"`
	OrderPublish          string `json:"order_publish"`
	EmailUpdateStatus     string `json:"email_update_status"`
	PublisherUpdateStatus string `json:"publisher_update_status"`
}

type ExchangeName struct {
	OrderEvent   string `json:"order_event"`
	PaymentEvent string `json:"payment_event"`
	UserEvent    string `json:"user_event"`
	ProductEvent string `json:"product_event"`
}

type QueueName struct {
	UpdatePaymentMethodDB string `json:"update_payment_method_db"`
	UpdatePaymentMethodES string `json:"update_payment_method_es"`
	UserSnapshot          string `json:"user_snapshot"`
	ProductSnapshot       string `json:"product_snapshot"`
}

type RabbitmqConfig struct {
	Host     string `json:"host" mapstructure:"RABBITMQ_HOST"`
	Port     string `json:"port" mapstructure:"RABBITMQ_PORT"`
	Username string `json:"username" mapstructure:"RABBITMQ_USER"`
	Password string `json:"password" mapstructure:"RABBITMQ_PASSWORD"`
}

type Elasticsearch struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	App           App            `json:"app"`
	Psql          PsqlDB         `json:"psql"`
	Redis         RedisConfig    `json:"redis"`
	RabbitMQ      RabbitmqConfig `json:"rabbitmq"`
	PublisherName PublisherName  `json:"publisher_name"`
	ExchangeName  ExchangeName   `json:"exchange_name"`
	QueueName     QueueName      `json:"queue_name"`
	Elasticsearch Elasticsearch  `json:"elasticsearch"`
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
			LatitudeRef:       viper.GetString("LATITUDE_REF"),
			LongitudeRef:      viper.GetString("LONGITUDE_REF"),
			MaxDistance:       viper.GetInt("MAX_DISTANCE"),
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
		PublisherName: PublisherName{
			ProductUpdateStock:    viper.GetString("PUBLISHER_PRODUCT_UPDATE_STOCK"),
			OrderPublish:          viper.GetString("PUBLISHER_ORDER"),
			EmailUpdateStatus:     viper.GetString("EMAIL_UPDATE_STATUS"),
			PublisherUpdateStatus: viper.GetString("PUBLISHER_UPDATE_STATUS"),
		},
		ExchangeName: ExchangeName{
			OrderEvent:   viper.GetString("EXCHANGE_ORDER_EVENT"),
			PaymentEvent: viper.GetString("EXCHANGE_PAYMENT_EVENT"),
			UserEvent:    viper.GetString("EXCHANGE_USER_EVENT"),
			ProductEvent: viper.GetString("EXCHANGE_PRODUCT_EVENT"),
		},
		QueueName: QueueName{
			UpdatePaymentMethodDB: viper.GetString("QUEUE_UPDATE_PAYMENT_METHOD_DB"),
			UpdatePaymentMethodES: viper.GetString("QUEUE_UPDATE_PAYMENT_METHOD_ES"),
			UserSnapshot:          viper.GetString("QUEUE_USER_SNAPSHOT_DB"),
			ProductSnapshot:       viper.GetString("QUEUE_PRODUCT_SNAPSHOT_DB"),
		},
		Elasticsearch: Elasticsearch{
			Host:     viper.GetString("ELASTICSEARCH_HOST"),
			Username: viper.GetString("ELASTICSEARCH_USERNAME"),
			Password: viper.GetString("ELASTICSEARCH_PASSWORD"),
		},
	}
}

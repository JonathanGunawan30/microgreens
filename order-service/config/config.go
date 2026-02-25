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
}

type PsqlDB struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
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
	ProductUpdateStock      string `json:"product_update_stock"`
	OrderPublish            string `json:"order_publish"`
	EmailUpdateStatus       string `json:"email_update_status"`
	PublisherPaymentSuccess string `json:"publisher_payment_success"`
	PublisherUpdateStatus   string `json:"publisher_update_status"`
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
		},
		Psql: PsqlDB{
			Host:      viper.GetString("DATABASE_HOST"),
			Port:      viper.GetString("DATABASE_PORT"),
			User:      viper.GetString("DATABASE_USER"),
			Password:  viper.GetString("DATABASE_PASSWORD"),
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
			ProductUpdateStock:      viper.GetString("PUBLISHER_PRODUCT_UPDATE_STOCK"),
			OrderPublish:            viper.GetString("PUBLISHER_ORDER"),
			EmailUpdateStatus:       viper.GetString("EMAIL_UPDATE_STATUS_NAME"),
			PublisherPaymentSuccess: viper.GetString("PUBLISHER_PAYMENT_SUCCESS"),
			PublisherUpdateStatus:   viper.GetString("PUBLISHER_UPDATE_STATUS"),
		},
		Elasticsearch: Elasticsearch{
			Host:     viper.GetString("ELASTICSEARCH_HOST"),
			Username: viper.GetString("ELASTICSEARCH_USERNAME"),
			Password: viper.GetString("ELASTICSEARCH_PASSWORD"),
		},
	}
}

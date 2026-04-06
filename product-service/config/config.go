package config

import "github.com/spf13/viper"

type App struct {
	AppPort string `json:"app_port"`
	AppEnv  string `json:"app_env"`

	JwtSecretKey string `json:"jwt_secret_key"`
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

type RabbitmqConfig struct {
	Host             string `json:"host" mapstructure:"RABBITMQ_HOST"`
	Port             string `json:"port" mapstructure:"RABBITMQ_PORT"`
	Username         string `json:"username" mapstructure:"RABBITMQ_USER"`
	Password         string `json:"password" mapstructure:"RABBITMQ_PASSWORD"`
	QueueStockUpdate string `json:"queue_stock_update" mapstructure:"RABBITMQ_QUEUE_STOCK_UPDATE"`
}

type Supabase struct {
	URL    string `json:"url"`
	Key    string `json:"key"`
	Bucket string `json:"bucket"`
}

type Elasticsearch struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type QueueName struct {
	ProductES string `json:"product_es"`
}

type ExchangeName struct {
	ProductEvent string `json:"product_event"`
}

type Config struct {
	App           App            `json:"app"`
	Psql          PsqlDB         `json:"psql"`
	Redis         RedisConfig    `json:"redis"`
	RabbitMQ      RabbitmqConfig `json:"rabbitmq"`
	Supabase      Supabase       `json:"supabase"`
	Elasticsearch Elasticsearch  `json:"elasticsearch"`
	ExchangeName  ExchangeName   `json:"exchange_name"`
	QueueName     QueueName      `json:"queue_name"`
}

func NewConfig() *Config {
	return &Config{
		App: App{
			AppPort:      viper.GetString("APP_PORT"),
			AppEnv:       viper.GetString("APP_ENV"),
			JwtSecretKey: viper.GetString("JWT_SECRET_KEY"),
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
			Host:             viper.GetString("RABBITMQ_HOST"),
			Port:             viper.GetString("RABBITMQ_PORT"),
			Username:         viper.GetString("RABBITMQ_USERNAME"),
			Password:         viper.GetString("RABBITMQ_PASSWORD"),
			QueueStockUpdate: viper.GetString("RABBITMQ_QUEUE_STOCK_UPDATE"),
		},
		Supabase: Supabase{
			URL:    viper.GetString("SUPABASE_STORAGE_URL"),
			Key:    viper.GetString("SUPABASE_STORAGE_KEY"),
			Bucket: viper.GetString("SUPABASE_STORAGE_BUCKET"),
		},
		Elasticsearch: Elasticsearch{
			Host:     viper.GetString("ELASTICSEARCH_HOST"),
			Username: viper.GetString("ELASTICSEARCH_USERNAME"),
			Password: viper.GetString("ELASTICSEARCH_PASSWORD"),
		},
		ExchangeName: ExchangeName{
			ProductEvent: viper.GetString("EXCHANGE_PRODUCT_EVENT"),
		},
		QueueName: QueueName{
			ProductES: viper.GetString("QUEUE_PRODUCT_ES"),
		},
	}
}

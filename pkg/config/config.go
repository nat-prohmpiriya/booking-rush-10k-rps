package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App             AppConfig       `mapstructure:"app"`
	Server          ServerConfig    `mapstructure:"server"`
	AuthDatabase    DatabaseConfig  `mapstructure:"auth_database"`    // Auth service database (required for auth-service)
	TicketDatabase  DatabaseConfig  `mapstructure:"ticket_database"`  // Ticket service database (required for ticket-service)
	BookingDatabase DatabaseConfig  `mapstructure:"booking_database"` // Booking service database
	PaymentDatabase DatabaseConfig  `mapstructure:"payment_database"` // Payment service database
	Redis           RedisConfig     `mapstructure:"redis"`
	Kafka           KafkaConfig     `mapstructure:"kafka"`
	MongoDB         MongoDBConfig   `mapstructure:"mongodb"`
	JWT             JWTConfig       `mapstructure:"jwt"`
	OTel            OTelConfig      `mapstructure:"otel"`
	Services        ServicesConfig  `mapstructure:"services"`
}

// ServicesConfig holds URLs of other microservices
type ServicesConfig struct {
	TicketServiceURL  string `mapstructure:"ticket_service_url"`
	AuthServiceURL    string `mapstructure:"auth_service_url"`
	PaymentServiceURL string `mapstructure:"payment_service_url"`
}

// AppConfig holds application-level settings
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"` // development, staging, production
	Debug       bool   `mapstructure:"debug"`
	Version     string `mapstructure:"version"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN returns the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Addr returns the Redis address
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaConfig holds Kafka/Redpanda connection settings
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	ClientID      string   `mapstructure:"client_id"`
}

// MongoDBConfig holds MongoDB connection settings
type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer           string        `mapstructure:"issuer"`
}

// OTelConfig holds OpenTelemetry settings
type OTelConfig struct {
	Enabled       bool    `mapstructure:"enabled"`
	ServiceName   string  `mapstructure:"service_name"`
	CollectorAddr string  `mapstructure:"collector_addr"`
	SampleRatio   float64 `mapstructure:"sample_ratio"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read from .env file (optional)
	if err := v.ReadInConfig(); err != nil {
		// It's okay if .env doesn't exist, we'll use environment variables
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only log if it's not a "file not found" error
			// We still continue because env vars might be set
		}
	}

	// Enable environment variable override
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	cfg := &Config{}
	if err := bindConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("failed to bind config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// LoadWithPath loads configuration from a specific path
func LoadWithPath(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("env")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	cfg := &Config{}
	if err := bindConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("failed to bind config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("APP_NAME", "booking-rush")
	v.SetDefault("APP_ENVIRONMENT", "development")
	v.SetDefault("APP_DEBUG", true)
	v.SetDefault("APP_VERSION", "1.0.0")

	// Server defaults
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_READ_TIMEOUT", "30s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "30s")
	v.SetDefault("SERVER_IDLE_TIMEOUT", "120s")

	// ==========================================================================
	// Per-Service Database Defaults (Microservice Architecture)
	// Each service MUST have its own database - no shared database allowed
	// ==========================================================================

	// Auth Database (auth-service)
	v.SetDefault("AUTH_DATABASE_HOST", "localhost")
	v.SetDefault("AUTH_DATABASE_PORT", 5432)
	v.SetDefault("AUTH_DATABASE_USER", "postgres")
	v.SetDefault("AUTH_DATABASE_PASSWORD", "postgres")
	v.SetDefault("AUTH_DATABASE_DBNAME", "auth_db")
	v.SetDefault("AUTH_DATABASE_SSLMODE", "disable")
	v.SetDefault("AUTH_DATABASE_MAX_OPEN_CONNS", 100)
	v.SetDefault("AUTH_DATABASE_MAX_IDLE_CONNS", 10)
	v.SetDefault("AUTH_DATABASE_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("AUTH_DATABASE_CONN_MAX_IDLE_TIME", "30m")

	// Ticket Database (ticket-service)
	v.SetDefault("TICKET_DATABASE_HOST", "localhost")
	v.SetDefault("TICKET_DATABASE_PORT", 5432)
	v.SetDefault("TICKET_DATABASE_USER", "postgres")
	v.SetDefault("TICKET_DATABASE_PASSWORD", "postgres")
	v.SetDefault("TICKET_DATABASE_DBNAME", "ticket_db")
	v.SetDefault("TICKET_DATABASE_SSLMODE", "disable")
	v.SetDefault("TICKET_DATABASE_MAX_OPEN_CONNS", 100)
	v.SetDefault("TICKET_DATABASE_MAX_IDLE_CONNS", 10)
	v.SetDefault("TICKET_DATABASE_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("TICKET_DATABASE_CONN_MAX_IDLE_TIME", "30m")

	// Booking Database (booking-service)
	v.SetDefault("BOOKING_DATABASE_HOST", "localhost")
	v.SetDefault("BOOKING_DATABASE_PORT", 5432)
	v.SetDefault("BOOKING_DATABASE_USER", "postgres")
	v.SetDefault("BOOKING_DATABASE_PASSWORD", "postgres")
	v.SetDefault("BOOKING_DATABASE_DBNAME", "booking_db")
	v.SetDefault("BOOKING_DATABASE_SSLMODE", "disable")
	v.SetDefault("BOOKING_DATABASE_MAX_OPEN_CONNS", 100)
	v.SetDefault("BOOKING_DATABASE_MAX_IDLE_CONNS", 10)
	v.SetDefault("BOOKING_DATABASE_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("BOOKING_DATABASE_CONN_MAX_IDLE_TIME", "30m")

	// Payment Database (payment-service)
	v.SetDefault("PAYMENT_DATABASE_HOST", "localhost")
	v.SetDefault("PAYMENT_DATABASE_PORT", 5432)
	v.SetDefault("PAYMENT_DATABASE_USER", "postgres")
	v.SetDefault("PAYMENT_DATABASE_PASSWORD", "postgres")
	v.SetDefault("PAYMENT_DATABASE_DBNAME", "payment_db")
	v.SetDefault("PAYMENT_DATABASE_SSLMODE", "disable")
	v.SetDefault("PAYMENT_DATABASE_MAX_OPEN_CONNS", 100)
	v.SetDefault("PAYMENT_DATABASE_MAX_IDLE_CONNS", 10)
	v.SetDefault("PAYMENT_DATABASE_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("PAYMENT_DATABASE_CONN_MAX_IDLE_TIME", "30m")

	// Redis defaults
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_POOL_SIZE", 100)
	v.SetDefault("REDIS_MIN_IDLE_CONNS", 10)
	v.SetDefault("REDIS_DIAL_TIMEOUT", "5s")
	v.SetDefault("REDIS_READ_TIMEOUT", "3s")
	v.SetDefault("REDIS_WRITE_TIMEOUT", "3s")

	// Kafka defaults
	v.SetDefault("KAFKA_BROKERS", "localhost:9092")
	v.SetDefault("KAFKA_CONSUMER_GROUP", "booking-rush")
	v.SetDefault("KAFKA_CLIENT_ID", "booking-rush")

	// MongoDB defaults
	v.SetDefault("MONGODB_URI", "mongodb://localhost:27017")
	v.SetDefault("MONGODB_DATABASE", "booking_rush")

	// JWT defaults
	v.SetDefault("JWT_SECRET", "your-secret-key-change-in-production")
	v.SetDefault("JWT_ACCESS_TOKEN_TTL", "15m")
	v.SetDefault("JWT_REFRESH_TOKEN_TTL", "168h") // 7 days
	v.SetDefault("JWT_ISSUER", "booking-rush")

	// OTel defaults
	v.SetDefault("OTEL_ENABLED", true)
	v.SetDefault("OTEL_SERVICE_NAME", "booking-rush")
	v.SetDefault("OTEL_COLLECTOR_ADDR", "localhost:4317")
	v.SetDefault("OTEL_SAMPLE_RATIO", 1.0)
}

func bindConfig(v *viper.Viper, cfg *Config) error {
	// App
	cfg.App.Name = v.GetString("APP_NAME")
	cfg.App.Environment = v.GetString("APP_ENVIRONMENT")
	cfg.App.Debug = v.GetBool("APP_DEBUG")
	cfg.App.Version = v.GetString("APP_VERSION")

	// Server
	cfg.Server.Host = v.GetString("SERVER_HOST")
	cfg.Server.Port = v.GetInt("SERVER_PORT")
	cfg.Server.ReadTimeout = v.GetDuration("SERVER_READ_TIMEOUT")
	cfg.Server.WriteTimeout = v.GetDuration("SERVER_WRITE_TIMEOUT")
	cfg.Server.IdleTimeout = v.GetDuration("SERVER_IDLE_TIMEOUT")

	// ==========================================================================
	// Per-Service Database Bindings (No fallback - true microservice)
	// ==========================================================================

	// Auth Database (auth-service)
	cfg.AuthDatabase.Host = v.GetString("AUTH_DATABASE_HOST")
	cfg.AuthDatabase.Port = v.GetInt("AUTH_DATABASE_PORT")
	cfg.AuthDatabase.User = v.GetString("AUTH_DATABASE_USER")
	cfg.AuthDatabase.Password = v.GetString("AUTH_DATABASE_PASSWORD")
	cfg.AuthDatabase.DBName = v.GetString("AUTH_DATABASE_DBNAME")
	cfg.AuthDatabase.SSLMode = v.GetString("AUTH_DATABASE_SSLMODE")
	cfg.AuthDatabase.MaxOpenConns = v.GetInt("AUTH_DATABASE_MAX_OPEN_CONNS")
	cfg.AuthDatabase.MaxIdleConns = v.GetInt("AUTH_DATABASE_MAX_IDLE_CONNS")
	cfg.AuthDatabase.ConnMaxLifetime = v.GetDuration("AUTH_DATABASE_CONN_MAX_LIFETIME")
	cfg.AuthDatabase.ConnMaxIdleTime = v.GetDuration("AUTH_DATABASE_CONN_MAX_IDLE_TIME")

	// Ticket Database (ticket-service)
	cfg.TicketDatabase.Host = v.GetString("TICKET_DATABASE_HOST")
	cfg.TicketDatabase.Port = v.GetInt("TICKET_DATABASE_PORT")
	cfg.TicketDatabase.User = v.GetString("TICKET_DATABASE_USER")
	cfg.TicketDatabase.Password = v.GetString("TICKET_DATABASE_PASSWORD")
	cfg.TicketDatabase.DBName = v.GetString("TICKET_DATABASE_DBNAME")
	cfg.TicketDatabase.SSLMode = v.GetString("TICKET_DATABASE_SSLMODE")
	cfg.TicketDatabase.MaxOpenConns = v.GetInt("TICKET_DATABASE_MAX_OPEN_CONNS")
	cfg.TicketDatabase.MaxIdleConns = v.GetInt("TICKET_DATABASE_MAX_IDLE_CONNS")
	cfg.TicketDatabase.ConnMaxLifetime = v.GetDuration("TICKET_DATABASE_CONN_MAX_LIFETIME")
	cfg.TicketDatabase.ConnMaxIdleTime = v.GetDuration("TICKET_DATABASE_CONN_MAX_IDLE_TIME")

	// Booking Database (booking-service)
	cfg.BookingDatabase.Host = v.GetString("BOOKING_DATABASE_HOST")
	cfg.BookingDatabase.Port = v.GetInt("BOOKING_DATABASE_PORT")
	cfg.BookingDatabase.User = v.GetString("BOOKING_DATABASE_USER")
	cfg.BookingDatabase.Password = v.GetString("BOOKING_DATABASE_PASSWORD")
	cfg.BookingDatabase.DBName = v.GetString("BOOKING_DATABASE_DBNAME")
	cfg.BookingDatabase.SSLMode = v.GetString("BOOKING_DATABASE_SSLMODE")
	cfg.BookingDatabase.MaxOpenConns = v.GetInt("BOOKING_DATABASE_MAX_OPEN_CONNS")
	cfg.BookingDatabase.MaxIdleConns = v.GetInt("BOOKING_DATABASE_MAX_IDLE_CONNS")
	cfg.BookingDatabase.ConnMaxLifetime = v.GetDuration("BOOKING_DATABASE_CONN_MAX_LIFETIME")
	cfg.BookingDatabase.ConnMaxIdleTime = v.GetDuration("BOOKING_DATABASE_CONN_MAX_IDLE_TIME")

	// Payment Database (payment-service)
	cfg.PaymentDatabase.Host = v.GetString("PAYMENT_DATABASE_HOST")
	cfg.PaymentDatabase.Port = v.GetInt("PAYMENT_DATABASE_PORT")
	cfg.PaymentDatabase.User = v.GetString("PAYMENT_DATABASE_USER")
	cfg.PaymentDatabase.Password = v.GetString("PAYMENT_DATABASE_PASSWORD")
	cfg.PaymentDatabase.DBName = v.GetString("PAYMENT_DATABASE_DBNAME")
	cfg.PaymentDatabase.SSLMode = v.GetString("PAYMENT_DATABASE_SSLMODE")
	cfg.PaymentDatabase.MaxOpenConns = v.GetInt("PAYMENT_DATABASE_MAX_OPEN_CONNS")
	cfg.PaymentDatabase.MaxIdleConns = v.GetInt("PAYMENT_DATABASE_MAX_IDLE_CONNS")
	cfg.PaymentDatabase.ConnMaxLifetime = v.GetDuration("PAYMENT_DATABASE_CONN_MAX_LIFETIME")
	cfg.PaymentDatabase.ConnMaxIdleTime = v.GetDuration("PAYMENT_DATABASE_CONN_MAX_IDLE_TIME")

	// Redis
	cfg.Redis.Host = v.GetString("REDIS_HOST")
	cfg.Redis.Port = v.GetInt("REDIS_PORT")
	cfg.Redis.Password = v.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = v.GetInt("REDIS_DB")
	cfg.Redis.PoolSize = v.GetInt("REDIS_POOL_SIZE")
	cfg.Redis.MinIdleConns = v.GetInt("REDIS_MIN_IDLE_CONNS")
	cfg.Redis.DialTimeout = v.GetDuration("REDIS_DIAL_TIMEOUT")
	cfg.Redis.ReadTimeout = v.GetDuration("REDIS_READ_TIMEOUT")
	cfg.Redis.WriteTimeout = v.GetDuration("REDIS_WRITE_TIMEOUT")

	// Kafka
	brokersStr := v.GetString("KAFKA_BROKERS")
	cfg.Kafka.Brokers = strings.Split(brokersStr, ",")
	cfg.Kafka.ConsumerGroup = v.GetString("KAFKA_CONSUMER_GROUP")
	cfg.Kafka.ClientID = v.GetString("KAFKA_CLIENT_ID")

	// MongoDB
	cfg.MongoDB.URI = v.GetString("MONGODB_URI")
	cfg.MongoDB.Database = v.GetString("MONGODB_DATABASE")

	// JWT
	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.JWT.AccessTokenTTL = v.GetDuration("JWT_ACCESS_TOKEN_TTL")
	cfg.JWT.RefreshTokenTTL = v.GetDuration("JWT_REFRESH_TOKEN_TTL")
	cfg.JWT.Issuer = v.GetString("JWT_ISSUER")

	// OTel
	cfg.OTel.Enabled = v.GetBool("OTEL_ENABLED")
	cfg.OTel.ServiceName = v.GetString("OTEL_SERVICE_NAME")
	cfg.OTel.CollectorAddr = v.GetString("OTEL_COLLECTOR_ADDR")
	cfg.OTel.SampleRatio = v.GetFloat64("OTEL_SAMPLE_RATIO")

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// Warn if using default JWT secret in production
	if c.App.Environment == "production" && c.JWT.Secret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}

	return nil
}

// ValidateAuthDatabase validates auth database configuration
func (c *Config) ValidateAuthDatabase() error {
	if c.AuthDatabase.Host == "" {
		return fmt.Errorf("AUTH_DATABASE_HOST is required")
	}
	if c.AuthDatabase.DBName == "" {
		return fmt.Errorf("AUTH_DATABASE_DBNAME is required")
	}
	return nil
}

// ValidateTicketDatabase validates ticket database configuration
func (c *Config) ValidateTicketDatabase() error {
	if c.TicketDatabase.Host == "" {
		return fmt.Errorf("TICKET_DATABASE_HOST is required")
	}
	if c.TicketDatabase.DBName == "" {
		return fmt.Errorf("TICKET_DATABASE_DBNAME is required")
	}
	return nil
}

// ValidateBookingDatabase validates booking database configuration
func (c *Config) ValidateBookingDatabase() error {
	if c.BookingDatabase.Host == "" {
		return fmt.Errorf("BOOKING_DATABASE_HOST is required")
	}
	if c.BookingDatabase.DBName == "" {
		return fmt.Errorf("BOOKING_DATABASE_DBNAME is required")
	}
	return nil
}

// ValidatePaymentDatabase validates payment database configuration
func (c *Config) ValidatePaymentDatabase() error {
	if c.PaymentDatabase.Host == "" {
		return fmt.Errorf("PAYMENT_DATABASE_HOST is required")
	}
	if c.PaymentDatabase.DBName == "" {
		return fmt.Errorf("PAYMENT_DATABASE_DBNAME is required")
	}
	return nil
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

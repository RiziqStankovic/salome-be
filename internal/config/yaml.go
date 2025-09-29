package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Server   ServerConfig   `yaml:"server"`
	Email    EmailConfig    `yaml:"email"`
	Redis    RedisConfig    `yaml:"redis"`
	Security SecurityConfig `yaml:"security"`
	Features FeatureConfig  `yaml:"features"`
	Midtrans MidtransConfig `yaml:"midtrans"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	Expiry string `yaml:"expiry"`
}

type ServerConfig struct {
	Port        int      `yaml:"port"`
	Host        string   `yaml:"host"`
	CORSOrigins []string `yaml:"cors_origins"`
}

type EmailConfig struct {
	SMTPHost string `yaml:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	FromName string `yaml:"from_name"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type SecurityConfig struct {
	BCryptCost    int    `yaml:"bcrypt_cost"`
	SessionSecret string `yaml:"session_secret"`
}

type FeatureConfig struct {
	EnableOTP               bool `yaml:"enable_otp"`
	EnableEmailVerification bool `yaml:"enable_email_verification"`
	EnablePaymentGateway    bool `yaml:"enable_payment_gateway"`
	EnableNotifications     bool `yaml:"enable_notifications"`
}

type MidtransConfig struct {
	ServerKey    string `yaml:"server_key"`
	ClientKey    string `yaml:"client_key"`
	BaseURL      string `yaml:"base_url"`
	IsProduction bool   `yaml:"is_production"`
}

var AppConfig *Config

func LoadConfig() error {
	// Try to find config file in different locations
	configPaths := []string{
		"secret/app.yaml",
		"app.yaml",
		"config/app.yaml",
		"./app.yaml",
	}

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return fmt.Errorf("config file not found in any of the expected locations: %v", configPaths)
	}

	// Read YAML file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", configPath, err)
	}

	// Parse YAML
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %v", err)
	}

	// Set default values if not specified
	setDefaults(config)

	AppConfig = config
	return nil
}

func setDefaults(config *Config) {
	// Database defaults
	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Database.User == "" {
		config.Database.User = "salome_user"
	}
	if config.Database.Password == "" {
		config.Database.Password = "salome_password_2024"
	}
	if config.Database.Name == "" {
		config.Database.Name = "salome_db"
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}

	// JWT defaults
	if config.JWT.Secret == "" {
		config.JWT.Secret = "salome-super-secret-jwt-key-2024-change-in-production"
	}
	if config.JWT.Expiry == "" {
		config.JWT.Expiry = "24h"
	}

	// Server defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if len(config.Server.CORSOrigins) == 0 {
		config.Server.CORSOrigins = []string{"http://localhost:3000"}
	}

	// Email defaults
	if config.Email.SMTPHost == "" {
		config.Email.SMTPHost = "smtp.gmail.com"
	}
	if config.Email.SMTPPort == 0 {
		config.Email.SMTPPort = 587
	}
	if config.Email.Username == "" {
		config.Email.Username = "your-email@gmail.com"
	}
	if config.Email.Password == "" {
		config.Email.Password = "your-app-password"
	}
	if config.Email.FromName == "" {
		config.Email.FromName = "SALOME Platform"
	}

	// Redis defaults
	if config.Redis.Host == "" {
		config.Redis.Host = "localhost"
	}
	if config.Redis.Port == 0 {
		config.Redis.Port = 6379
	}
	if config.Redis.Password == "" {
		config.Redis.Password = ""
	}
	if config.Redis.DB == 0 {
		config.Redis.DB = 0
	}

	// Security defaults
	if config.Security.BCryptCost == 0 {
		config.Security.BCryptCost = 12
	}
	if config.Security.SessionSecret == "" {
		config.Security.SessionSecret = "salome-session-secret-2024"
	}

	// Feature defaults
	config.Features.EnableOTP = true
	config.Features.EnableEmailVerification = true
	config.Features.EnablePaymentGateway = true
	config.Features.EnableNotifications = true
}

func GetConfig() *Config {
	if AppConfig == nil {
		// Try to load config if not already loaded
		if err := LoadConfig(); err != nil {
			// If loading fails, create a default config
			config := &Config{}
			setDefaults(config)
			AppConfig = config
		}
	}
	return AppConfig
}

// Helper functions for backward compatibility
func GetEnv(key, defaultValue string) string {
	config := GetConfig()

	switch key {
	case "DB_HOST":
		return config.Database.Host
	case "DB_PORT":
		return fmt.Sprintf("%d", config.Database.Port)
	case "DB_USER":
		return config.Database.User
	case "DB_PASSWORD":
		return config.Database.Password
	case "DB_NAME":
		return config.Database.Name
	case "DB_SSLMODE":
		return config.Database.SSLMode
	case "JWT_SECRET":
		return config.JWT.Secret
	case "JWT_EXPIRY":
		return config.JWT.Expiry
	case "PORT":
		return fmt.Sprintf("%d", config.Server.Port)
	case "CORS_ORIGIN":
		if len(config.Server.CORSOrigins) > 0 {
			return config.Server.CORSOrigins[0]
		}
		return "http://localhost:3000"
	case "SMTP_HOST":
		return config.Email.SMTPHost
	case "SMTP_PORT":
		return fmt.Sprintf("%d", config.Email.SMTPPort)
	case "SMTP_USERNAME":
		return config.Email.Username
	case "SMTP_PASSWORD":
		return config.Email.Password
	case "SMTP_FROM_NAME":
		return config.Email.FromName
	case "REDIS_HOST":
		return config.Redis.Host
	case "REDIS_PORT":
		return fmt.Sprintf("%d", config.Redis.Port)
	case "REDIS_PASSWORD":
		return config.Redis.Password
	case "REDIS_DB":
		return fmt.Sprintf("%d", config.Redis.DB)
	case "BCRYPT_COST":
		return fmt.Sprintf("%d", config.Security.BCryptCost)
	case "SESSION_SECRET":
		return config.Security.SessionSecret
	case "ENABLE_OTP":
		return fmt.Sprintf("%t", config.Features.EnableOTP)
	case "ENABLE_EMAIL_VERIFICATION":
		return fmt.Sprintf("%t", config.Features.EnableEmailVerification)
	case "ENABLE_PAYMENT_GATEWAY":
		return fmt.Sprintf("%t", config.Features.EnablePaymentGateway)
	case "ENABLE_NOTIFICATIONS":
		return fmt.Sprintf("%t", config.Features.EnableNotifications)
	default:
		return defaultValue
	}
}

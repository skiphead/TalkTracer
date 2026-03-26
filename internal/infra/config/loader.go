package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure.
type Config struct {
	Bot          Bot            `yaml:"bot"`
	Telegram     TelegramConfig `yaml:"telegram"`
	SaluteSpeech OAuth          `yaml:"salute_speech"`
	GigaChat     OAuth          `yaml:"giga_chat"`
	DBConfig     DBConfig       `yaml:"db_config"`
}

// Bot contains bot-specific configuration.
type Bot struct {
	WorkerCount int    `yaml:"worker_count"`
	TmpDir      string `yaml:"tmp_dir"`
}

// TelegramConfig contains Telegram Bot API settings.
type TelegramConfig struct {
	Address        string        `yaml:"address"`
	Insecure       bool          `yaml:"insecure"`
	Token          string        `yaml:"token"`
	PollInterval   time.Duration `yaml:"poll_interval"`
	ParseMode      string        `yaml:"parse_mode"`
	AllowedUpdates []string      `yaml:"allowed_updates"`
}

// SaluteSpeechConfig contains Salute Speech API configuration.
type SaluteSpeechConfig struct {
	OAuth OAuth `yaml:"oauth"`
}

// GigaChatConfig contains GigaChat API configuration.
type GigaChatConfig struct {
	OAuth OAuth `yaml:"oauth"`
}

// OAuth contains OAuth 2.0 client credentials.
type OAuth struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// DBConfig contains database connection settings.
type DBConfig struct {
	Schema   string `yaml:"schema"`
	DBName   string `yaml:"dbname"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
}

// Load loads and parses the configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if cfg.DBConfig.Port == 0 {
		cfg.DBConfig.Port = 5432
	}

	cfg.setDefaults()

	return &cfg, nil
}

// Validate checks required configuration fields.
func (c *Config) Validate() error {
	if c.Telegram.Token == "" {
		return fmt.Errorf("telegram.token is required")
	}

	if c.DBConfig.DBName == "" {
		return fmt.Errorf("db_config.dbname is required")
	}
	if c.DBConfig.User == "" {
		return fmt.Errorf("db_config.user is required")
	}
	if c.DBConfig.Password == "" {
		return fmt.Errorf("db_config.password is required")
	}
	if c.DBConfig.Host == "" {
		return fmt.Errorf("db_config.host is required")
	}

	return nil
}

// setDefaults sets default values for configuration fields.
func (c *Config) setDefaults() {
	if c.Telegram.Address == "" {
		c.Telegram.Address = "https://api.telegram.org"
	}
	if c.Telegram.PollInterval == 0 {
		c.Telegram.PollInterval = 3 * time.Second
	}
	if c.Telegram.ParseMode == "" {
		c.Telegram.ParseMode = telebot.ModeHTML
	}
	if c.Telegram.AllowedUpdates == nil {
		c.Telegram.AllowedUpdates = []string{
			"message", "edited_message", "callback_query",
		}
	}
}

// GetBaseURL returns the base URL for Telegram Bot API requests.
func (t *TelegramConfig) GetBaseURL() string {
	addr := t.Address
	if addr == "" {
		addr = "https://api.telegram.org"
	}
	addr = strings.TrimSuffix(addr, "/api")
	addr = strings.TrimSuffix(addr, "/")
	return fmt.Sprintf("%s/bot%s", addr, t.Token)
}

// ToTelebotSettings converts the configuration to telebot settings.
func (t *TelegramConfig) ToTelebotSettings() telebot.Settings {
	return telebot.Settings{
		URL:       t.Address,
		Token:     t.Token,
		ParseMode: t.ParseMode,
		Poller: &telebot.LongPoller{
			Timeout:        t.PollInterval,
			AllowedUpdates: t.AllowedUpdates,
		},
	}
}

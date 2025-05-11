package config

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const EnvFileName = ".env"

// Config структура для парсинга файла конфигурации
type Config struct {
	Server ServerConfig `yaml:"server"`
}

// ServerConfig структура для парсинга файла конфигурации
type ServerConfig struct {
	Host    string        `yaml:"host"`
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// NewConfig парсит данные из файла конфигурации и возвращает объект Config
func NewConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}

// FetchConfigPath возвращает путь к файлу конфигурации
func FetchConfigPath() string {
	err := godotenv.Load(filepath.Join("..", EnvFileName))
	if err != nil {
		log.Println("No .env file found, using default config path")
	}

	// Если в .env файле нет переменной CONFIG_PATH, используем путь по умолчанию
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join("..", "config.yml")
	}

	return configPath
}

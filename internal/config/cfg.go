package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config хранит конфигурацию сервера
type Config struct {
	ServerAddress string // Адрес запуска HTTP-сервера
	DBconnStr     string // Строка с адресом подключения к БД
}

// NewConfig создает и инициализирует конфигурацию из аргументов командной строки
func NewConfig() *Config {
	cfg := &Config{}

	defaultServerAddr := "localhost:8080"

	// Парсим флаги
	serverAddr := flag.String("a", defaultServerAddr, "Адрес запуска HTTP-сервера")
	dbStr := flag.String("d", "", "Строка с адресом подключения к БД")
	flag.Parse()

	var ok bool

	// Берём адрес из переменной окружения
	cfg.ServerAddress, ok = os.LookupEnv("RUN_ADDRESS")
	if !ok {
		// Если адреса нет, определяем флаги
		if *serverAddr == "" {
			*serverAddr = defaultServerAddr
		}

		// Устанавливаем значение
		cfg.ServerAddress = *serverAddr
	}

	// Берём настройки БД из переменной окружения
	cfg.DBconnStr, ok = os.LookupEnv("DATABASE_URI")
	if !ok {
		// Берём настроейк БД нет, определяем флаги
		cfg.DBconnStr = *dbStr
	}

	return cfg
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	c.ServerAddress = strings.ReplaceAll(c.ServerAddress, " ", "")
	c.DBconnStr = strings.ReplaceAll(c.DBconnStr, " ", "")

	if c.ServerAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.DBconnStr == "" {
		return fmt.Errorf("DB conn str cannot be empty")
	}
	return nil
}

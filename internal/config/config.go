package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/logger"
	"go.uber.org/zap"
)

const (
	DefaultHost = "localhost:8080"
	DefaultURL  = "http://localhost:8080"
)

// Config — единая структура для всех источников
type Config struct {
	ServerAddress   string `json:"server_address" env:"SERVER_ADDRESS"`
	BaseURL         string `json:"base_url" env:"BASE_URL"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `json:"database_dsn" env:"DATABASE_DSN"`
	AuditFile       string `json:"-" env:"AUDIT_FILE"`
	AuditURL        string `json:"-" env:"AUDIT_URL"`
	EnableHTTPS     bool   `json:"enable_https" env:"ENABLE_HTTPS"`
	ConfigPath      string `json:"-" env:"CONFIG"`
	TrustedSubnet   string `json:"trusted_subnet" env:"TRUSTED_SUBNET"`
}

type SettingsObject struct {
	Server1       Server
	Server2       Server
	Log           *zap.Logger
	DatabaseDSN   string
	FilePath      string
	AuditFile     string
	AuditURL      string
	EnableHTTPS   bool
	TrustedSubnet string
}

type Server struct {
	Addr    string
	BaseURL string
}

func Settings() SettingsObject {
	logger.Initialize("info")

	// 1. Конфигурация из Флагов
	flagCfg := parseFlags()

	// 2. Конфигурация из ENV
	envCfg := parseEnv()

	// 3. Конфигурация из JSON (если путь указан)
	configPath := flagCfg.ConfigPath
	if envCfg.ConfigPath != "" {
		configPath = envCfg.ConfigPath
	}

	jsonCfg := parseJSON(configPath)

	// Итоговая сборка с помощью mergo.
	// Приоритет (от низшего к высшему): JSON -> ENV -> Flags
	finalCfg := jsonCfg

	// Накладываем ENV на JSON
	if err := mergo.Merge(&finalCfg, envCfg, mergo.WithOverride); err != nil {
		logger.Log.Error(fmt.Sprintf("Mergo error (ENV): %v", err))
	}

	// Накладываем Flags на результат (флаги имеют высший приоритет, если они установлены)
	// Для корректной работы mergo с флагами, в parseFlags нужно возвращать только заполненные значения
	if err := mergo.Merge(&finalCfg, flagCfg, mergo.WithOverride); err != nil {
		logger.Log.Error(fmt.Sprintf("Mergo error (Flags): %v", err))
	}

	// Дефолтные значения, если всё пусто
	if finalCfg.ServerAddress == "" {
		finalCfg.ServerAddress = DefaultHost
	}
	if finalCfg.BaseURL == "" {
		finalCfg.BaseURL = "http://" + finalCfg.ServerAddress
	}

	return SettingsObject{
		Server1:       Server{Addr: finalCfg.ServerAddress, BaseURL: finalCfg.BaseURL},
		Server2:       Server{Addr: finalCfg.ServerAddress, BaseURL: finalCfg.BaseURL},
		Log:           logger.Log,
		DatabaseDSN:   finalCfg.DatabaseDSN,
		FilePath:      finalCfg.FileStoragePath,
		AuditFile:     finalCfg.AuditFile,
		AuditURL:      finalCfg.AuditURL,
		EnableHTTPS:   finalCfg.EnableHTTPS,
		TrustedSubnet: finalCfg.TrustedSubnet,
	}
}

func parseFlags() Config {
	var c Config
	// Используем временные переменные, чтобы mergo не затер пустые строки дефолтами флагов
	serverAddress1 := flag.String("a", "", "значение может быть таким: "+DefaultHost+"|"+DefaultURL)
	serverAddress2 := flag.String("b", "", "значение может быть таким: "+DefaultHost+"|"+DefaultURL)
	dsn := flag.String("d", "", "реквизиты базы данных")
	file := flag.String("f", "", "путь к файлу для хранения данных")
	aFile := flag.String("audit-file", "", "путь к файлу-приёмнику, в который сохраняются логи аудита")
	aURL := flag.String("audit-url", "", "полный URL удаленного сервера-приёмника, куда отправляются логи аудита")
	trustedSubnet := flag.String("t", "", "доверенная подсеть")
	conf := flag.String("c", "", "Файл конфигурации")
	flag.StringVar(conf, "config", "", "Файл конфигурации")
	enableHTTPS := flag.Bool("s", false, "Enable HTTPS")

	flag.Parse()

	c.ServerAddress = *serverAddress1
	c.BaseURL = *serverAddress2
	c.DatabaseDSN = *dsn
	c.FileStoragePath = *file
	c.ConfigPath = *conf
	c.AuditFile = *aFile
	c.AuditURL = *aURL
	c.TrustedSubnet = *trustedSubnet

	// С bool сложнее: флаг всегда false по умолчанию.
	// Проверяем, был ли он явно передан в командной строке.
	if isFlagPassed("s") {
		c.EnableHTTPS = *enableHTTPS
	}

	return c
}

func parseEnv() Config {
	return Config{
		ServerAddress:   os.Getenv("SERVER_ADDRESS"),
		BaseURL:         os.Getenv("BASE_URL"),
		DatabaseDSN:     os.Getenv("DATABASE_DSN"),
		FileStoragePath: os.Getenv("FILE_STORAGE_PATH"),
		ConfigPath:      os.Getenv("CONFIG"),
		AuditFile:       os.Getenv("AUDIT_FILE"),
		AuditURL:        os.Getenv("AUDIT_URL"),
		EnableHTTPS:     os.Getenv("ENABLE_HTTPS") == "true",
		TrustedSubnet:   os.Getenv("TRUSTED_SUBNET"),
	}
}

func parseJSON(path string) Config {
	var c Config

	if path == "" {
		return c
	}

	file, err := os.Open(path)

	if err != nil {
		return c
	}

	defer file.Close()
	json.NewDecoder(file).Decode(&c)

	return c
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})

	return found
}

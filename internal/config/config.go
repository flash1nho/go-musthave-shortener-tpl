package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/logger"

	"go.uber.org/zap"
)

const (
	DefaultHost = "localhost:8080"
	DefaultURL  = "http://localhost:8080"
)

type Server struct {
	Addr    string
	BaseURL string
}

type NetAddress struct {
	Host string
	Port int
}

type Config struct {
	ServerAddress string `json:"server_address"`
	BaseURL       string `json:"base_url"`
	FilePath      string `json:"file_storage_path"`
	DatabaseDSN   string `json:"database_dsn"`
	EnableHTTPS   bool   `json:"enable_https"`
}

func (addr *NetAddress) String() string {
	return addr.Host + ":" + strconv.Itoa(addr.Port)
}

func (addr *NetAddress) Set(s string) error {
	trimmed := strings.TrimPrefix(s, "http://")
	hp := strings.Split(trimmed, ":")

	if len(hp) != 2 {
		return errors.New("значение может быть таким: " + DefaultHost + "|" + DefaultURL)
	}

	port, err := strconv.Atoi(hp[1])

	if err != nil {
		return err
	}

	addr.Host = hp[0]
	addr.Port = port

	return nil
}

func Settings() (Server, Server, *zap.Logger, string, string, string, string, *bool) {
	serverAddress1 := new(NetAddress)
	_ = flag.Value(serverAddress1)
	flag.Var(serverAddress1, "a", "значение может быть таким: "+DefaultHost+"|"+DefaultURL)

	serverAddress2 := new(NetAddress)
	_ = flag.Value(serverAddress2)
	flag.Var(serverAddress2, "b", "значение может быть таким: "+DefaultHost+"|"+DefaultURL)

	var databaseDSN, filePath, auditFile, auditURL, ConfigPath string

	flag.StringVar(&databaseDSN, "d", "", "реквизиты базы данных")
	flag.StringVar(&filePath, "f", "", "путь к файлу для хранения данных")
	flag.StringVar(&auditFile, "audit-file", "", "путь к файлу-приёмнику, в который сохраняются логи аудита")
	flag.StringVar(&auditURL, "audit-url", "", "полный URL удаленного сервера-приёмника, куда отправляются логи аудита")
	flag.StringVar(&ConfigPath, "c", "", "Файл конфигурации")
	flag.StringVar(&ConfigPath, "config", "", "Файл конфигурации")

	enableHTTPS := flag.Bool("s", false, "Поддержка SSL")

	flag.Parse()

	envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN")

	if ok {
		databaseDSN = envDatabaseDSN
	}

	envPath, ok := os.LookupEnv("FILE_STORAGE_PATH")

	if ok {
		filePath = envPath
	}

	envAuditFile, ok := os.LookupEnv("AUDIT_FILE")

	if ok {
		auditFile = envAuditFile
	}

	envAuditURL, ok := os.LookupEnv("AUDIT_URL")

	if ok {
		auditURL = envAuditURL
	}

	envConfigPath, ok := os.LookupEnv("CONFIG")

	if ok {
		ConfigPath = envConfigPath
	}

	envEnableHTTPS, ok := os.LookupEnv("ENABLE_HTTPS")

	if ok && envEnableHTTPS == "true" {
		*enableHTTPS = true
	}

	logger.Initialize("info")

	if ConfigPath != "" {
		file, err := os.Open(ConfigPath)

		if err != nil {
			logger.Log.Error(fmt.Sprint(err))
		}

		defer file.Close()

		var fileCfg Config

		if err := json.NewDecoder(file).Decode(&fileCfg); err != nil {
			logger.Log.Error(fmt.Sprint(err))
		}

		_, ok = os.LookupEnv("SERVER_ADDRESS")

		if !ok && !isFlagPassed("a") && fileCfg.ServerAddress != "" {
			serverAddress1.Set(fileCfg.ServerAddress)
		}

		_, ok = os.LookupEnv("BASE_URL")

		if !ok && !isFlagPassed("b") && fileCfg.BaseURL != "" {
			serverAddress2.Set(fileCfg.BaseURL)
		}

		if filePath == "" && fileCfg.FilePath != "" {
			filePath = fileCfg.FilePath
		}

		if databaseDSN == "" && fileCfg.DatabaseDSN != "" {
			databaseDSN = fileCfg.DatabaseDSN
		}

		if !*enableHTTPS && fileCfg.EnableHTTPS {
			*enableHTTPS = fileCfg.EnableHTTPS
		}
	}

	return ServerData(serverAddress1.String()),
		ServerData(serverAddress2.String()),
		logger.Log,
		databaseDSN,
		filePath,
		auditFile,
		auditURL,
		enableHTTPS
}

func ServerData(serverAddress string) Server {
	envServerAddress, ok := os.LookupEnv("SERVER_ADDRESS")

	if ok {
		serverAddress = envServerAddress
	} else if serverAddress == ":0" {
		serverAddress = DefaultHost
	}

	trimmedServerAddress := strings.TrimPrefix(serverAddress, "http://")
	serverBaseURL := "http://" + trimmedServerAddress

	envBaseURL, ok := os.LookupEnv("BASE_URL")

	if ok {
		serverBaseURL = envBaseURL
	}

	return Server{Addr: serverAddress, BaseURL: serverBaseURL}
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

package config

import (
    "flag"
    "fmt"
    "errors"
    "strings"
    "strconv"
    "os"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/logger"

    "go.uber.org/zap"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
    DefaultHost = "localhost:8080"
    DefaultURL = "http://localhost:8080"
    DefaultFilePath = ""
    DefaultDatabaseDSN = ""
)

type Server struct {
    Addr string
    BaseURL string
}

type NetAddress struct {
    Host string
    Port int
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

func Settings() (Server, Server, *zap.Logger, string, string) {
    serverAddress1 := new(NetAddress)
    _ = flag.Value(serverAddress1)
    flag.Var(serverAddress1, "a", "значение может быть таким: " + DefaultHost + "|" + DefaultURL)

    serverAddress2 := new(NetAddress)
    _ = flag.Value(serverAddress2)
    flag.Var(serverAddress2, "b", "значение может быть таким: " + DefaultHost + "|" + DefaultURL)

    var databaseDSN string
    flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "реквизиты базы данных")

    var filePath string
    flag.StringVar(&filePath, "f", DefaultFilePath, "путь к файлу для хранения данных")

    flag.Parse()

    if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
        databaseDSN = envDatabaseDSN
    }

    if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
        filePath = envPath
    }

    logger.Initialize("info")

    runMigrations(databaseDSN, logger.Log)

    return ServerData(fmt.Sprint(serverAddress1)),
           ServerData(fmt.Sprint(serverAddress2)),
           logger.Log,
           databaseDSN,
           filePath
}

func ServerData(serverAddress string) Server {
    if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
        serverAddress = envServerAddress
    } else if serverAddress == ":0" {
        serverAddress = DefaultHost
    }

    serverBaseURL := "http://" + serverAddress

    if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
        serverBaseURL = envBaseURL
    }

    return Server{Addr: serverAddress, BaseURL: serverBaseURL}
}

func runMigrations(databaseDSN string, log *zap.Logger) {
    if databaseDSN != "" {
        m, err := migrate.New("file://migrations", databaseDSN)

        if err != nil {
            log.Fatal(fmt.Sprintf("Ошибка загрузки миграций: %v", err))
        }

        if err := m.Up(); err != nil && err != migrate.ErrNoChange {
            log.Fatal(fmt.Sprintf("Ошибка запуска миграций: %v", err))
        }
    }
}

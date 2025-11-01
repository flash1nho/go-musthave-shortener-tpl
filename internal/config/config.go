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
)

const (
    DefaultHost = "localhost:8080"
    DefaultURL = "http://localhost:8080"
    DefaultFilePath = "/tmp/shorten.json"
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

func Settings() (Server, Server, string, *zap.Logger) {
    serverAddress1 := new(NetAddress)
    _ = flag.Value(serverAddress1)
    flag.Var(serverAddress1, "a", "значение может быть таким: " + DefaultHost + "|" + DefaultURL)

    serverAddress2 := new(NetAddress)
    _ = flag.Value(serverAddress2)
    flag.Var(serverAddress2, "b", "значение может быть таким: " + DefaultHost + "|" + DefaultURL)

    var filePath string
    flag.StringVar(&filePath, "f", DefaultFilePath, "путь к файлу для хранения данных")

    flag.Parse()

    if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
        filePath = envPath
    }

    logger.Initialize("info")

    return ServerData(fmt.Sprint(serverAddress1)), ServerData(fmt.Sprint(serverAddress2)), filePath, logger.Log
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

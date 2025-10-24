package config

import (
    "flag"
    "fmt"
    "errors"
    "strings"
    "strconv"
    "os"
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
    hp := strings.Split(s, ":")

    if len(hp) != 2 {
        return errors.New("значение может быть таким: localhost:8080")
    }

    port, err := strconv.Atoi(hp[1])

    if err != nil {
        return err
    }

    addr.Host = hp[0]
    addr.Port = port

    return nil
}

func Servers() (Server, Server) {
    serverAddress1 := new(NetAddress)
    _ = flag.Value(serverAddress1)
    flag.Var(serverAddress1, "a", "значение может быть таким: localhost:8080")

    serverAddress2 := new(NetAddress)
    _ = flag.Value(serverAddress2)
    flag.Var(serverAddress2, "b", "значение может быть таким: localhost:8080")

    flag.Parse()

    return ServerData(fmt.Sprint(serverAddress1)), ServerData(fmt.Sprint(serverAddress2))
}

func ServerData(serverAddress string) Server {
    if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
        serverAddress = envServerAddress
    } else if serverAddress == ":0" {
        serverAddress = "localhost:8080"
    }

    serverBaseURL := "http://" + serverAddress

    if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
        serverBaseURL = envBaseURL
    }

    return Server{Addr: serverAddress, BaseURL: serverBaseURL}
}

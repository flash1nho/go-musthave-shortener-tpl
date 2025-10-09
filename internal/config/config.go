package config

import (
    "flag"
    "fmt"
    "errors"
    "strings"
    "strconv"
)

type NetAddress struct {
    Host string
    Port int
}

func (a NetAddress) String() string {
    return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *NetAddress) Set(s string) error {
    hp := strings.Split(s, ":")
    if len(hp) != 2 {
        return errors.New("значение может быть таким: localhost:8888")
    }
    port, err := strconv.Atoi(hp[1])
    if err != nil{
        return err
    }
    a.Host = hp[0]
    a.Port = port
    return nil
}

func Servers() (string, string) {
    a := new(NetAddress)
    _ = flag.Value(a)
    flag.Var(a, "a", "значение может быть таким: localhost:8888")

    b := new(NetAddress)
    _ = flag.Value(b)
    flag.Var(b, "b", "значение может быть таким: localhost:8888")

    flag.Parse()

    return hostWithPort(fmt.Sprint(a)), hostWithPort(fmt.Sprint(b))
}

func hostWithPort(host string) (string) {
    if host == ":0" {
        return "localhost:8080"
    }

    return host
}

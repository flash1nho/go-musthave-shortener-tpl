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

func (addr *NetAddress) String() string {
    return addr.Host + ":" + strconv.Itoa(addr.Port)
}

func (addr *NetAddress) Set(s string) error {
    hp := strings.Split(s, ":")

    if len(hp) != 2 {
        return errors.New("значение может быть таким: localhost:8888")
    }

    port, err := strconv.Atoi(hp[1])

    if err != nil {
        return err
    }

    addr.Host = hp[0]
    addr.Port = port

    return nil
}

func Hosts() (string, string) {
    host1 := new(NetAddress)
    _ = flag.Value(host1)
    flag.Var(host1, "a", "значение может быть таким: localhost:8888")

    host2 := new(NetAddress)
    _ = flag.Value(host2)
    flag.Var(host2, "b", "значение может быть таким: localhost:8888")

    flag.Parse()

    return hostWithPort(fmt.Sprint(host1)), hostWithPort(fmt.Sprint(host2))
}

func hostWithPort(host string) string {
    if host == ":0" {
        return "localhost:8080"
    }

    return host
}

package main

import (
	  "go-musthave-shortener-tpl/internal/config"
	  "go-musthave-shortener-tpl/internal/router"
)

func main() {
	  a, b := config.Servers()
	  router.Start(a, b)
}

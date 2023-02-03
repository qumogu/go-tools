package main

import (
	"github.com/qumogu/go-tools/example/server"
)

func main() {
	svr := server.NewServer()
	svr.Init()
	svr.Run()
}

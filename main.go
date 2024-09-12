package main

import (
	"test/pkg/server"
)

func main() {
	gameServer := server.NewGame()
	gameServer.Start()
}

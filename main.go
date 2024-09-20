package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"test/pkg/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	f, err := os.Create("cpu.pprof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	gameServer := server.NewGame()
	gameServer.Start(ctx)
}

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/porn2oautobuyer/app"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	buyer, err := app.New(*configPath)
	if err != nil {
		log.Fatalf("failed to create autobuyer: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := buyer.Start(ctx); err != nil {
		log.Fatalf("autobuyer exited with error: %v", err)
	}
}

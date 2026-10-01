package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

var configPath = flag.String("config", "gqlgen.yml", "Path to gqlgen.yml")

func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, *configPath)
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gqlexec: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"log"
	"os"

	"github.com/jmuk/sylvan/pkg/chat"
)

func main() {
	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	c, err := chat.New(ctx, cwd)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if err := c.RunLoop(ctx); err != nil {
		log.Print(err)
	}
}

package main

import (
	"log"

	"ducklogs/internal/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		log.Fatal(err)
	}
}

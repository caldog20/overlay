package main

import (
	"log"

	"github.com/caldog20/overlay/controller/cmd"
)

func main() {
	log.Println("Starting Controller...")
	cmd.Execute()
}

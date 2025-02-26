package main

import (
	"os"

	backend "github.com/staticbackendhq/core"
)

func main() {
	dbHost := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8099"
	}

	backend.Start(dbHost, port)
}

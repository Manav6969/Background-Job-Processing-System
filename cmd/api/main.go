package main

import (
	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/server"
)

func main() {
	cfg := config.Load()
	err := db.Connect("postgres://postgres@localhost:5432/backgroundjobprocessingsystem")
	if err != nil {
		panic(err)
	}
	server.Run(cfg)
}

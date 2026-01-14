package main

import (
	"github.com/Manav6969/Background-Job-Processing-System/internal/config"
	"github.com/Manav6969/Background-Job-Processing-System/internal/db"
	"github.com/Manav6969/Background-Job-Processing-System/internal/server"
	"github.com/Manav6969/Background-Job-Processing-System/internal/job"
	"os"
	
)

func main() {
	cfg := config.Load()
	err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	job.InitQueue()
	server.Run(cfg)
}

package main

import (
    "github.com/Manav6969/Background-Job-Processing-System/internal/config"
    "github.com/Manav6969/Background-Job-Processing-System/internal/server"
)

func main() {
    cfg := config.Load()
    server.Run(cfg)
}

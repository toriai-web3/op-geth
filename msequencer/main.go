package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/msequencer/server"
)

const port = 8888

func main() {
	ser := server.NewServer(port)
	log.Info("start server", "port", port)
	_ = ser.Start()
}

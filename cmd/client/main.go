package main

import (
	"slava/internal/slava/client"
	"slava/pkg/logger"
)

func main() {
	ch := make(chan int)
	_, err := client.MakeClient("127.0.0.1:6399")
	if err != nil {
		logger.Error("make client error:", err)
	}
	<-ch
}

package main

import (
	"fmt"
	"os"

	"slava/config"
	. "slava/internal/data"
	"slava/internal/slava/server"
	"slava/internal/tcp"
	"slava/pkg/logger"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func main() {
	print(Banner)
	logger.Setup(&logger.Settings{
		Path:       "logs",
		Name:       "slava",
		Ext:        "log",
		TimeFormat: "2006-01-02",
	})
	configFilename := os.Getenv("CONFIG")
	if configFilename == "" {
		if fileExists("slava.conf") {
			config.SetupConfig("slava.conf")
		} else {
			config.Properties = DefaultProperties
		}
	} else {
		config.SetupConfig(configFilename)
	}

	err := tcp.ListenAndServeWithSignal(&tcp.Config{
		Address: fmt.Sprintf("%s:%d", config.Properties.Bind, config.Properties.Port),
	}, server.MakeHandler())
	if err != nil {
		logger.Error(err)
	}
}

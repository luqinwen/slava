package data

import "slava/config"

var Banner = `
          ██                             
         ░██                             
  ██████ ░██  ██████   ██    ██  ██████  
 ██░░░░  ░██ ░░░░░░██ ░██   ░██ ░░░░░░██ 
░░█████  ░██  ███████ ░░██ ░██   ███████ 
 ░░░░░██ ░██ ██░░░░██  ░░████   ██░░░░██ 
 ██████  ███░░████████  ░░██   ░░████████
░░░░░░  ░░░  ░░░░░░░░    ░░     ░░░░░░░░ 

`

var DefaultProperties = &config.ServerProperties{
	Bind:           "0.0.0.0",
	Port:           6399,
	AppendOnly:     false,
	AppendFilename: "",
	MaxClients:     1000,
}

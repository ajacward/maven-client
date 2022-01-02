package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ajacward/maven-client/maven"
)

var config Config

const usage = `usage: %s
Retrieve compile/runtime maven group artifact version list based on repo URL and input file

Options:
`

func main() {
	flag.StringVar(&config.configPath, "config", "config.txt", "Configuration")
	flag.StringVar(&config.inputPath, "input", "input.txt", "List of dependencies")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	config.load()

	dependencies := make(map[string]maven.Project)

	for _, coordinate := range config.readInput() {
		queryCoordinate(coordinate, config, dependencies)
	}

	writeCoordinates(dependencies)
}

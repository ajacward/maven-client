package main

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ajacward/maven-client/file"
)

type Config struct {
	configPath string
	inputPath  string
	repoUrl    string
	userName   string
	password   string
}

func (c *Config) load() {
	filePtr, _ := file.Open(c.configPath)
	defer filePtr.Close()

	scanner := bufio.NewScanner(filePtr)

	for scanner.Scan() {
		property := strings.Split(scanner.Text(), "=")
		if len(property) == 2 {
			name := strings.ToLower(strings.TrimSpace(property[0]))
			value := strings.TrimSpace(property[1])

			switch name {
			case "repourl":
				config.repoUrl = value
			case "username":
				config.userName = value
			case "password":
				config.password = value
			default:
				fmt.Println("Unrecognized config property", name)
			}
		} else {
			fmt.Println("Expected name = value, read", property[0])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
}

func (c *Config) readInput() []string {
	filePtr, _ := file.Open(c.inputPath)
	defer filePtr.Close()

	coordinates := make([]string, 0, 5)

	scanner := bufio.NewScanner(filePtr)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Count(line, ":") == 2 {
			coordinates = append(coordinates, line)
		} else {
			fmt.Println("Invalid GAV coordinate:", line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	return coordinates
}

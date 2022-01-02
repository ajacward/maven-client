package file

import (
	"log"
	"os"
)

func Open(path string) (*os.File, error) {
	filePtr, err := os.Open(path)

	if err != nil {
		log.Fatalf("Failed to open %s", path)
	}

	return filePtr, err
}

func Create(path string) (*os.File, error) {
	filePtr, err := os.Create("output.txt")

	if err != nil {
		log.Fatalf("Failed to create file %s", path)
	}

	return filePtr, err
}

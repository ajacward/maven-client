package main

import (
	"bufio"
	"fmt"
	"sort"

	"github.com/ajacward/maven-client/file"
	"github.com/ajacward/maven-client/maven"
)

func writeCoordinates(dependencies map[string]maven.Project) {
	filePtr, _ := file.Create("output.txt")
	defer filePtr.Close()

	w := bufio.NewWriter(filePtr)

	keys := make([]string, len(dependencies))

	var i int

	for k := range dependencies {
		keys[i] = k
		i += 1
	}

	sort.Strings(keys)

	for _, k := range keys {
		fmt.Fprintln(w, k)
	}

	w.Flush()
}

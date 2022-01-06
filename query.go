package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/ajacward/maven-client/maven"
	"golang.org/x/text/encoding/ianaindex"
)

func defaultString(value string, defaultValue string) string {
	result := value

	if len(value) == 0 {
		result = defaultValue
	}

	return result
}

func buildProps(project maven.Project, deps map[string]maven.Project) map[string]string {
	props := make(map[string]string, 0)

	if len(project.Parent.ArtifactId) > 0 {
		props = buildProps(deps[project.Parent.GavForm()], deps)
		props["project.parent.version"] = project.Parent.Version
	}

	for k, v := range project.Properties {
		props[k] = v
	}

	props["project.version"] = project.Version

	r, _ := regexp.Compile(`\$\{(.+)\}`)

	for k, v := range props {
		if strings.Contains(v, "${") {
			token := r.FindStringSubmatch(v)[1]

			props[k] = props[token]
		}
	}

	return props
}

func Unmarshal(data []byte, v interface{}) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = func(charset string, reader io.Reader) (io.Reader, error) {
		enc, err := ianaindex.IANA.Encoding(charset)
		if err != nil {
			return nil, fmt.Errorf("charset %s: %s", charset, err.Error())
		}
		if enc == nil {
			// Assume it's compatible with (a subset of) UTF-8 encoding
			// Bug: https://github.com/golang/go/issues/19421
			return reader, nil
		}
		return enc.NewDecoder().Reader(reader), nil
	}

	return decoder.Decode(v)
}

func queryCoordinate(coordinate string, config Config, dependencies map[string]maven.Project) {
	_, prs := dependencies[coordinate]

	if !prs {
		d := maven.NewMavenCoordinate(coordinate)
		url := fmt.Sprintf("%s/%s.pom", config.repoUrl, d.UrlPath())
		fmt.Println(url)
		resp, err := http.Get(url)

		if err != nil {
			log.Fatalln(err)
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatalln(err)
		}

		var project maven.Project

		err = Unmarshal(body, &project)

		if err != nil {
			log.Fatalln("error:", err)
		}

		dependencies[coordinate] = project

		if len(project.Parent.ArtifactId) > 0 {
			queryCoordinate(project.Parent.GavForm(), config, dependencies)
		}

		props := buildProps(project, dependencies)

		for _, dm := range project.DependencyManagement.Dependencies {
			if dm.Scope == "import" {
				r, _ := regexp.Compile(`\$\{(.+)\}`)

				if strings.Contains(dm.GroupId, "${") {
					groupToken := r.FindStringSubmatch(dm.GroupId)[1]
					dm.GroupId = props[groupToken]
				}

				if strings.Contains(dm.Version, "${") {
					versionToken := r.FindStringSubmatch(dm.Version)[1]
					dm.Version = props[versionToken]
				}

				queryCoordinate(dm.GavForm(), config, dependencies)
			}
		}
	}
}

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

func findInheritedValue(dep maven.Dependency, parent maven.Project, deps map[string]maven.Project) maven.Dependency {
	var inheritedValue maven.Dependency

	for _, d := range parent.DependencyManagement.Dependencies.Dependency {
		if d.ArtifactId == dep.ArtifactId && d.GroupId == dep.GroupId {
			inheritedValue = d
			break
		}
	}

	if len(inheritedValue.ArtifactId) == 0 && len(parent.Parent.ArtifactId) > 0 {
		inheritedValue = findInheritedValue(dep, deps[parent.Parent.GavForm()], deps)
	}

	return inheritedValue
}

func findInheritedValues(dep maven.Dependency, parent maven.Project, deps map[string]maven.Project) maven.Dependency {
	inheritedValues := findInheritedValue(dep, parent, deps)

	props := buildProps(parent, deps)

	r, _ := regexp.Compile(`\$\{(.+)\}`)

	if strings.Contains(inheritedValues.Version, "${") {
		versionToken := r.FindStringSubmatch(inheritedValues.Version)[1]
		inheritedValues.Version = props[versionToken]
	}

	return inheritedValues
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

		parent := dependencies[project.Parent.GavForm()]

		props := buildProps(project, dependencies)

		for _, dependency := range project.Dependencies.Dependency {
			inheritedValues := findInheritedValues(dependency, parent, dependencies)

			if dependency.Scope == "" {
				dependency.Scope = defaultString(inheritedValues.Scope, "compile")
			}

			if dependency.Scope == "compile" || dependency.Scope == "runtime" {
				r, _ := regexp.Compile(`\$\{(.+)\}`)

				if strings.Contains(dependency.GroupId, "${") {
					groupToken := r.FindStringSubmatch(dependency.GroupId)[1]
					dependency.GroupId = props[groupToken]
				}

				if strings.Contains(dependency.Version, "${") {
					versionToken := r.FindStringSubmatch(dependency.Version)[1]
					dependency.Version = props[versionToken]
				}

				queryCoordinate(
					fmt.Sprintf(
						"%s:%s:%s",
						defaultString(dependency.GroupId, inheritedValues.GroupId),
						dependency.ArtifactId,
						defaultString(dependency.Version, inheritedValues.Version)),
					config, dependencies)
			}
		}
	}
}

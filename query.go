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

	for _, d := range parent.DependencyManagement.Dependencies {
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

func any(exclusions []maven.Exclusion, f func(maven.Exclusion) bool) bool {
	exists := false

	for _, exclusion := range exclusions {
		if f(exclusion) {
			exists = true
			break
		}
	}

	return exists
}

func queryCoordinate(coordinate string, exclusions []maven.Exclusion, config Config, dependencies map[string]maven.Project) {
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
			queryCoordinate(project.Parent.GavForm(), []maven.Exclusion{}, config, dependencies)
		}

		parent := dependencies[project.Parent.GavForm()]

		props := buildProps(project, dependencies)

		if len(exclusions) == 0 {
			exclusions = findInheritedValues(maven.Dependency{GroupId: project.GroupId, ArtifactId: project.ArtifactId}, parent, dependencies).Exclusions
		}

		bomDependencyManagement := make([]maven.Dependency, 0)

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

				queryCoordinate(dm.GavForm(),
					[]maven.Exclusion{},
					config, dependencies)

				bomProps := buildProps(dependencies[dm.GavForm()], dependencies)

				for _, bmDm := range dependencies[dm.GavForm()].DependencyManagement.Dependencies {
					if strings.Contains(bmDm.Version, "${") {
						versionToken := r.FindStringSubmatch(bmDm.Version)[1]
						bmDm.Version = bomProps[versionToken]
					}

					bomDependencyManagement = append(bomDependencyManagement, bmDm)
				}
			}
		}

		for _, dependency := range project.Dependencies {
			inheritedValues := findInheritedValues(dependency, project, dependencies)

			if dependency.Version == "" {
				for _, bomDm := range bomDependencyManagement {
					if bomDm.GroupId == dependency.GroupId && bomDm.ArtifactId == dependency.ArtifactId {
						dependency.Version = bomDm.Version
						break
					}
				}
			}

			if dependency.Scope == "" {
				dependency.Scope = defaultString(inheritedValues.Scope, "compile")
			}

			if !dependency.Optional {
				dependency.Optional = inheritedValues.Optional
			}

			isExcluded := any(exclusions, func(e maven.Exclusion) bool {
				return e.GroupId == dependency.GroupId && e.ArtifactId == dependency.ArtifactId
			})

			if !isExcluded && (dependency.Scope == "compile" || dependency.Scope == "runtime") && !dependency.Optional {
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
						defaultString(defaultString(dependency.Version, inheritedValues.Version), d.Version)),
					dependency.Exclusions,
					config, dependencies)
			}
		}
	}
}

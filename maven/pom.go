package maven

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type mavenCoordinate struct {
	Group    string
	Artifact string
	Version  string
}

func (mc *mavenCoordinate) UrlPath() string {
	return fmt.Sprintf("%[1]s/%[2]s/%[3]s/%[2]s-%[3]s", strings.ReplaceAll(mc.Group, ".", "/"), mc.Artifact, mc.Version)
}

func NewMavenCoordinate(gav string) mavenCoordinate {
	coords := strings.Split(gav, ":")

	mc := mavenCoordinate{
		Group:    coords[0],
		Artifact: coords[1],
		Version:  coords[2],
	}

	return mc
}

type Project struct {
	Parent               Parent               `xml:"parent"`
	GroupId              string               `xml:"groupId"`
	ArtifactId           string               `xml:"artifactId"`
	Version              string               `xml:"version"`
	Dependencies         []Dependency         `xml:"dependencies>dependency"`
	DependencyManagement DependencyManagement `xml:"dependencyManagement"`
	Properties           Properties           `xml:"properties"`
}

func (p *Project) GavForm() string {
	return fmt.Sprintf("%s:%s:%s", p.GroupId, p.ArtifactId, p.Version)
}

type Parent struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
}

func (p *Parent) GavForm() string {
	return fmt.Sprintf("%s:%s:%s", p.GroupId, p.ArtifactId, p.Version)
}

type Dependency struct {
	GroupId    string      `xml:"groupId"`
	ArtifactId string      `xml:"artifactId"`
	Version    string      `xml:"version"`
	Scope      string      `xml:"scope"`
	Optional   bool        `xml:"optional"`
	Exclusions []Exclusion `xml:"exclusions>exclusion"`
}

type Exclusion struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
}

func (d *Dependency) GavForm() string {
	return fmt.Sprintf("%s:%s:%s", d.GroupId, d.ArtifactId, d.Version)
}

type DependencyManagement struct {
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

type Properties map[string]string

func (p *Properties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*p = map[string]string{}
	for {
		key := ""
		value := ""
		token, err := d.Token()

		if err == io.EOF {
			break
		}

		switch tokenType := token.(type) {
		case xml.StartElement:
			key = tokenType.Name.Local
			err := d.DecodeElement(&value, &start)

			if err != nil {
				return err
			}

			(*p)[key] = value
		}
	}

	return nil
}

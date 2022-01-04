package maven

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type mavenCoordinate struct {
	group    string
	artifact string
	version  string
}

func (mc *mavenCoordinate) UrlPath() string {
	return fmt.Sprintf("%[1]s/%[2]s/%[3]s/%[2]s-%[3]s", strings.ReplaceAll(mc.group, ".", "/"), mc.artifact, mc.version)
}

func NewMavenCoordinate(gav string) mavenCoordinate {
	coords := strings.Split(gav, ":")

	mc := mavenCoordinate{
		group:    coords[0],
		artifact: coords[1],
		version:  coords[2],
	}

	return mc
}

type Project struct {
	Parent               Parent               `xml:"parent"`
	GroupId              string               `xml:"groupId"`
	ArtifactId           string               `xml:"artifactId"`
	Version              string               `xml:"version"`
	Dependencies         Dependencies         `xml:"dependencies"`
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

type Dependencies struct {
	Dependency []Dependency `xml:"dependency"`
}

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   bool   `xml:"optional"`
}

func (d *Dependency) GavForm() string {
	return fmt.Sprintf("%s:%s:%s", d.GroupId, d.ArtifactId, d.Version)
}

type DependencyManagement struct {
	Dependencies Dependencies `xml:"dependencies"`
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

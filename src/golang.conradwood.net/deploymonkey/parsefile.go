package main

import (
	"errors"
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type FileDef struct {
	Namespace string
	Groups    []*pb.GroupDefinitionRequest
}

func ParseFile(fname string) (*FileDef, error) {
	fmt.Printf("Parsing %s\n", fname)
	fb, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Printf("Failed to read file %s: %s\n", fname, err)
		return nil, err
	}
	res, err := ParseConfig(fb)
	if err != nil {
		fmt.Printf("Failed to parse file %s: %s\n", fname, err)
		return nil, err
	}
	fmt.Printf("Found %d groups in file %s\n", len(res.Groups), fname)
	fmt.Printf("Namespace: %s\n", res.Namespace)
	for _, x := range res.Groups {
		PrintGroup(x)
	}

	return res, nil
}
func ParseConfig(config []byte) (*FileDef, error) {
	gd := FileDef{}
	err := yaml.Unmarshal(config, &gd)
	if err != nil {
		fmt.Printf("Failed to parse yaml: %s\n", err)
		return nil, err
	}
	// apply namespace throughout
	for _, x := range gd.Groups {
		if x.Namespace == "" {
			x.Namespace = gd.Namespace
		}
	}
	for _, g := range gd.Groups {
		for _, app := range g.Applications {
			err = CheckAppComplete(app)
			if err != nil {
				return nil, err
			}
		}
	}
	return &gd, nil
}
func CheckAppComplete(app *pb.ApplicationDefinition) error {
	s := fmt.Sprintf("%s-%s", app.Repository, app.DeploymentID)
	if app.DownloadURL == "" {
		return errors.New(fmt.Sprintf("%s is invalid: Missing DownloadURL", s))
	}
	return nil
}

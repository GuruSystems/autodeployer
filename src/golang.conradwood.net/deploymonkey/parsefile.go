package main

import (
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type FileDef struct {
	Namespace string
	Groups    []pb.GroupDefinitionRequest
}

func ParseFile(fname string) (*FileDef, error) {
	fmt.Printf("Parsing %s\n", fname)
	fb, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Printf("Failed to read file %s: %s\n", fname, err)
		return nil, err
	}
	gd := FileDef{}
	err = yaml.Unmarshal(fb, &gd)
	if err != nil {
		fmt.Printf("Failed to parse file %s: %s\n", fname, err)
		return nil, err
	}
	// apply namespace throughout
	for _, x := range gd.Groups {
		if x.Namespace == "" {
			x.Namespace = gd.Namespace
		}
	}
	fmt.Printf("Found %d groups in file %s\n", len(gd.Groups), fname)
	fmt.Printf("Namespace: %s\n", gd.Namespace)
	for _, x := range gd.Groups {
		fmt.Printf("  Group: %s with %d applications\n", x.GroupID, len(x.Applications))
		for _, a := range x.Applications {
			fmt.Printf("        Application: \n")
			fmt.Printf("           Repo  : %s\n", a.Repository)
			fmt.Printf("           Binary: %s\n", a.Binary)
		}
	}
	return &gd, nil
}

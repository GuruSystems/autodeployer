package main

import (
	"errors"
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
)

type AppDiff struct {
	Was pb.ApplicationDefinition
	Is  pb.ApplicationDefinition
}
type Diff struct {
	AppDiffs []AppDiff
}

// describe in human terms whas this diff represents!
func (ad *AppDiff) Describe() string {
	s := fmt.Sprintf("%v -> %v", ad.Was, ad.Is)
	return s
}

// compare groupdefinition request and work out the differences
func Compare(def1, def2 *pb.GroupDefinitionRequest) (*Diff, error) {

	// 1. find all applications that exist in def1 but not def2
	// 2. find all applications that exist in def2 but not def1
	// 3. find all applications that exist in def1 and def2 but are different
	return nil, errors.New("not implemented")
}

package main

import (
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
)

type AppDiff struct {
	Was *pb.ApplicationDefinition
	Is  *pb.ApplicationDefinition
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
	diff := &Diff{}
	// 1. find all applications that exist in def1 but not def2
	findNonExists(def1, def2, false, diff)
	// 2. find all applications that exist in def2 but not def1
	findNonExists(def2, def1, true, diff)
	// 3. find all applications that exist in def1 and def2 but are different
	// TODO

	fmt.Printf("Found %d differences:\n", len(diff.AppDiffs))
	for _, x := range diff.AppDiffs {
		fmt.Printf("Diff: %s\n", x.Describe())
	}
	return diff, nil
}

// add all that exist in def1 but not in def2 to diff
func findNonExists(def1, def2 *pb.GroupDefinitionRequest, reverse bool, diff *Diff) error {
	for _, ad := range def1.Applications {
		if !doesExistInDef(ad, def2.Applications) {
			nad := AppDiff{}
			if reverse {
				nad.Is = ad
			} else {
				nad.Was = ad
			}
			diff.AppDiffs = append(diff.AppDiffs, nad)
		}
	}
	return nil
}

func doesExistInDef(ad *pb.ApplicationDefinition, ads []*pb.ApplicationDefinition) bool {
	for _, x := range ads {
		if isSame(ad, x) {
			return true
		}
	}
	return false
}

func isSame(ad1, ad2 *pb.ApplicationDefinition) bool {
	if ad1.Repository != ad2.Repository {
		return false
	}
	return true
}

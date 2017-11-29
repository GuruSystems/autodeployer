package main

import (
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
)

func PrintGroup(x *pb.GroupDefinitionRequest) {
	fmt.Printf("  Group: %s with %d applications\n", x.GroupID, len(x.Applications))
	fmt.Printf("        Namespace  : %s\n", x.Namespace)
	for _, a := range x.Applications {
		fmt.Printf("        Application: \n")
		fmt.Printf("           Repo  : %s\n", a.Repository)
		fmt.Printf("           Binary: %s\n", a.Binary)
		fmt.Printf("          BuildID: %d\n", a.BuildID)
	}
}

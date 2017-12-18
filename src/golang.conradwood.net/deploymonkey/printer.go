package main

import (
	"bytes"
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
)

func PrintGroup(x *pb.GroupDefinitionRequest) {
	var b bytes.Buffer
	fmt.Printf("  Group: %s with %d applications\n", x.GroupID, len(x.Applications))
	fmt.Printf("        Namespace  : %s\n", x.Namespace)
	for _, a := range x.Applications {
		fmt.Printf("        Application: \n")
		fmt.Printf("           Repo  : %s\n", a.Repository)
		fmt.Printf("           Binary: %s\n", a.Binary)
		fmt.Printf("          BuildID: %d\n", a.BuildID)
		b.WriteString(fmt.Sprintf("           %d Args: ", len(a.Args)))
		for _, arg := range a.Args {
			b.WriteString(fmt.Sprintf("%s ", arg))
		}
		b.WriteString("%\n")
		for _, autoreg := range a.AutoRegs {
			b.WriteString(fmt.Sprintf("           Autoregistration: "))
			b.WriteString(fmt.Sprintf("%s ", autoreg))
		}
		fmt.Printf("%s\n", b.String())
	}
}

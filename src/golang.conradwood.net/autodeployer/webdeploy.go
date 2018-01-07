package main

import (
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
)

// this deploys a webpackage
// it runs within the autodeployer-server process space

func DeployWebPackage(cr *pb.DeployRequest) error {
	fmt.Printf("Deploying webpackage: %v\n", cr)
	return nil
}

package main

import (
	"flag"
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"os"
)

var (
	webdir = flag.String("webdir", "/var/www/static", "Web directory to deploy webpackages into")
)

// this deploys a webpackage
// it runs within the autodeployer-server process space
// this *should* be async?
func DeployWebPackage(cr *pb.DeployRequest) error {
	fmt.Printf("Deploying webpackage: %v\n", cr)
	targetdir := *webdir
	os.Mkdir(targetdir, 0777)

	// BAD BAD BAD - this is ONE process space so this might overlap
	// change to my working directory
	err := os.Chdir(targetdir)
	if err != nil {
		fmt.Printf("Failed to Chdir() to %s: %s\n", targetdir, err)
	}
	target := "WEB.TAR"
	err = DownloadBinary(cr.DownloadURL, target, cr.DownloadUser, cr.DownloadPassword)
	if err != nil {
		fmt.Printf("Failed to download url %s: %s\n", cr.DownloadURL, err)
		return err
	}
	return nil
}

package main

import (
	"flag"
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"os"
	"sync"
)

var (
	webdir  = flag.String("webdir", "/var/www/static", "Web directory to deploy webpackages into")
	webLock = sync.Mutex
)

// this deploys a webpackage
// it runs within the autodeployer-server process space
// this *should* be async?
func DeployWebPackage(cr *pb.DeployRequest) error {
	fmt.Printf("Deploying webpackage: %v\n", cr)

	// this lock is here because I do not know what will
	// happen if two threads call Chdir(). I suspect
	// (because they are threads not processes) it'll
	// change the other threads' working dir ;(
	webLock.Lock()
	defer webLock.Unlock()

	targetdir := fmt.Sprintf("%s/%s", *webdir, cr.Repository)
	os.Mkdir(targetdir, 0777)

	// BAD BAD BAD - this is ONE process space so this might overlap
	// when we enter this in another thread
	// change to my working directory
	err := os.Chdir(targetdir)
	if err != nil {
		fmt.Printf("Failed to Chdir() to %s: %s\n", targetdir, err)
	}
	target := "" // ignored for tarfiles anyways
	err = DownloadBinary(cr.DownloadURL, target, cr.DownloadUser, cr.DownloadPassword)
	if err != nil {
		fmt.Printf("Failed to download url %s: %s\n", cr.DownloadURL, err)
		return err
	}
	return nil
}

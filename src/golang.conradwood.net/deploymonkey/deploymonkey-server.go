package main

// this is the main, privileged daemon. got to run as root because we're forking off
// different users from here

// fileaccess is split out to starter.go, which runs as an unprivileged user

// (this is done by virtue of exec('su',Args[0]) )
// the flag msgid goes into the startup code, so do not run the privileged daemon with that flag!

import (
	"errors"
	"flag"
	"fmt"
	pb "golang.conradwood.net/deploymonkey/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// static variables for flag parser
var (
	port = flag.Int("port", 4000, "The server port")
)

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(DeployMonkey)
	// Register the handler object
	pb.RegisterDeployMonkeyServer(server, s)
	return nil
}

func main() {
	flag.Parse() // parse stuff. see "var" section above

	sd := server.ServerDef{
		Port: *port,
	}
	sd.Register = st
	err := server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
	}
	fmt.Printf("Done\n")
	return

}

/**********************************
* implementing the functions here:
***********************************/
type DeployMonkey struct {
	wtf int
}

func (s *DeployMonkey) DefineGroup(ctx context.Context, cr *pb.GroupDefinitionRequest) (*pb.EmptyResponse, error) {

	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
}

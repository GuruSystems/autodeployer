package main

import (
	"fmt"
	"golang.conradwood.net/client"

	sb "gitlab.gurusys.co.uk/guru/proto/slackgateway"
	pb "golang.conradwood.net/deploymonkey/proto"
)

func NotifyPeopleAboutDeploy(dbgroup *DBGroup, apps []*pb.ApplicationDefinition, version int) {
	conn, err := client.DialWrapper("slackgateway.SlackGateway")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()
	cl := sb.NewSlackGatewayClient(conn)
	msg := fmt.Sprintf("Applied %d of %s to the datacenter, containing: \n", version, dbgroup.groupDef.Namespace)
	for _, app := range apps {
		msg := msg + fmt.Sprintf("   %d instances: build #%d of application %s\n", app.Instances, app.BuildID, app.Binary)
	}
	pm := &sb.PublishMessageRequest{OriginService: "originservicenotfilledinyet",
		Channel: "deployments",
		Test:    msg,
	}
	_, err = cl.PublishMessage(ctx, pm)
	if err != nil {
		fmt.Printf("Failed to post slack message: %s\n", err)
	} else {
		fmt.Printf("Posted slack message: %s\n", msg)
	}

}

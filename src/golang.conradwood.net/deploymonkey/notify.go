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
	pm := &sb.PublishMessageRequest{OriginService: "originservicenotfilledinyet",
		Channel: "deployments",
		Test:    fmt.Sprintf("Deployed version %d of %s", dbgroup.groupDef.Namespace),
	}
	_, err := cl.PublishMessage(ctx, pm)
	if err != nil {
		fmt.Printf("Failed to post slack message: %s\n", err)
	}

}

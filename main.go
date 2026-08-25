package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

// GroupName is the API group cert-manager uses to reach this solver.
var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified (e.g. acme.stackryze.com)")
	}
	// Runs the webhook apiserver and serves the DNS01 solver.
	cmd.RunWebhookServer(GroupName, &stackryzeSolver{})
}

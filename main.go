package main

import (
	"context"
	"flag"
	"log"

	provider "terraform-provider-looker/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "1.0.0"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debug")
	flag.Parse()

	err := providerserver.Serve(
		context.Background(),
		provider.New(version),
		providerserver.ServeOpts{

			Address: "registry.terraform.io/techbytes09/looker",
			Debug:   debug,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}

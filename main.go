package main

import (
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
	"github.com/sharkyze/waypoint-plugin-gae/platform"
	"github.com/sharkyze/waypoint-plugin-gae/release"
)

func main() {
	// sdk.Main allows you to register the components which should
	// be included in your plugin
	// Main sets up all the go-plugin requirements
	sdk.Main(sdk.WithComponents(&platform.Platform{}, &release.ReleaseManager{}))
}

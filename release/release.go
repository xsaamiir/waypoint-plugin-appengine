package release

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"google.golang.org/api/appengine/v1"

	"github.com/sharkyze/waypoint-plugin-appengine/internal/appengineutil"
	"github.com/sharkyze/waypoint-plugin-appengine/platform"
)

type ReleaseConfig struct{}

type ReleaseManager struct {
	config ReleaseConfig
}

// Config implements component.Configurable.
func (rm *ReleaseManager) Config() (interface{}, error) {
	return &rm.config, nil
}

// ConfigSet implements component.ConfigurableNotify.
func (rm *ReleaseManager) ConfigSet(config interface{}) error {
	_, ok := config.(*ReleaseConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit.
		return fmt.Errorf("Expected *ReleaseConfig as parameter")
	}

	// validate the config

	return nil
}

// ReleaseFunc implements component.ReleaseManager.
func (rm *ReleaseManager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.release
}

// A BuildFunc does not have a strict signature, you can define the parameters
// you need based on the Available parameters that the Waypoint SDK provides.
// Waypoint will automatically inject parameters as specified
// in the signature at run time.
//
// Available input parameters:
// - context.Context
// - *component.Source
// - *component.JobInfo
// - *component.DeploymentConfig
// - *datadir.Project
// - *datadir.App
// - *datadir.Component
// - hclog.Logger
// - terminal.UI
// - *component.LabelSet
//
// In addition to default input parameters the platform.Deployment from the Deploy step
// can also be injected.
//
// The output parameters for ReleaseFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *ReleaseManager) release(
	ctx context.Context,
	deployment *platform.Deployment,
	ui terminal.UI,
) (*Release, error) {
	st := ui.Status()
	defer st.Close()

	project := deployment.Project
	service := deployment.Service
	versionID := deployment.VersionId

	st.Update("Releasing App Engine version '" + versionID + "'")

	appengineService, err := appengine.NewService(ctx)
	if err != nil {
		return nil, err
	}

	servicePatchCall := appengineService.Apps.Services.Patch(project, service, &appengine.Service{
		Split: &appengine.TrafficSplit{Allocations: map[string]float64{versionID: 1}},
	})
	servicePatchCall.UpdateMask("split")
	servicePatchCall = servicePatchCall.Context(ctx)

	op, err := servicePatchCall.Do()
	if err != nil {
		return nil, err
	}

	op, err = appengineutil.WaitForOperation(ctx, appengineService, op)
	if err != nil {
		return nil, err
	}

	if op.Error != nil {
		st.Step(terminal.StatusError, "Traffic split error")
		return nil, errors.New(op.Error.Message)
	}

	st.Step(terminal.StatusOK, "Traffic split successful")

	return &Release{}, nil
}

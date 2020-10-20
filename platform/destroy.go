package platform

import (
	"context"
	"errors"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"google.golang.org/api/appengine/v1"
)

// DestroyFunc implements the Destroyer interface.
func (p *Platform) DestroyFunc() interface{} {
	return p.destroy
}

// A DestroyFunc does not have a strict signature, you can define the parameters
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
// In addition to default input parameters the Deployment from the DeployFunc step
// can also be injected.
//
// The output parameters for PushFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) destroy(ctx context.Context, ui terminal.UI, deployment *Deployment) error {
	st := ui.Status()
	defer st.Close()

	projectID := deployment.ProjectId
	service := deployment.Service
	versionID := deployment.VersionId

	st.Update(
		"Deleting App Engine version '" +
			"apps/" + projectID + "/services/" + service + "/versions/" + versionID +
			"'",
	)

	appengineService, err := appengine.NewService(ctx)
	if err != nil {
		return err
	}

	deleteCall := appengineService.Apps.Services.Versions.Delete(projectID, service, versionID)

	deleteCall = deleteCall.Context(ctx)
	op, err := deleteCall.Do()
	if err != nil {
		st.Step(terminal.StatusError, "Error deleting App Engine version")
		return err
	}

	op, err = waitForOperation(ctx, appengineService, op)
	if err != nil {
		st.Step(terminal.StatusError, "Error fetching delete operation status")
		return err
	}

	if op.Error != nil {
		st.Step(terminal.StatusError, "Error deleting App Engine version")
		return errors.New(op.Error.Message)
	}

	return nil
}

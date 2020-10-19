package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/sharkyze/waypoint-plugin-gcs/registry"
	"google.golang.org/api/appengine/v1"
)

type DeployConfig struct {
	Application string `hcl:"application"`
	Service     string `hcl:"service"`
	// Runtime: Desired runtime. Example: go114.
	Runtime string `hcl:"runtime"`
	// InstanceClass: Instance class that is used to run this version. Valid
	// values are: AutomaticScaling: F1, F2, F4, F4_1G ManualScaling or
	// BasicScaling: B1, B2, B4, B8, B4_1GDefaults to F1 for
	// AutomaticScaling and B1 for ManualScaling or BasicScaling.
	InstanceClass string            `hcl:"instance_class,optional"`
	EnvVars       map[string]string `hcl:"environment_variables,optional"`
	RuntimeMainExecutablePath string `hcl:main,optional`
}

type Platform struct {
	config DeployConfig
}

// Config implements Configurable.
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// ConfigSet jmplements ConfigurableNotify.
func (p *Platform) ConfigSet(config interface{}) error {
	c, ok := config.(*DeployConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("Expected *DeployConfig as parameter")
	}

	// validate the config
	if c.Runtime == "" {
		return errors.New("Runtime should not be empty")
	}

	if c.Service == "" {
		return errors.New("Service should not be empty")
	}

	return nil
}

// DeployFunc implements Builder.
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
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

// In addition to default input parameters the registry.Artifact from the Build step
// can also be injected.
//
// The output parameters for BuildFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) deploy(ctx context.Context, artifact *registry.Artifact, ui terminal.UI) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Update("Deploying application '" + artifact.Source + "'")

	appengineService, err := appengine.NewService(ctx)
	if err != nil {
		return nil, err
	}

	service := p.config.Service
	appID := p.config.Application
	versionID := time.Now().Format("20060102t150405")
	aev := appengine.Version{
		ApiConfig:           nil,
		AutomaticScaling:    nil,
		BasicScaling:        nil,
		BetaSettings:        nil,
		BuildEnvVariables:   nil,
		DefaultExpiration:   "",
		Deployment:          &appengine.Deployment{Zip: &appengine.ZipInfo{SourceUrl: artifact.Source}},
		DiskUsageBytes:      0,
		EndpointsApiService: nil,
		Entrypoint:          nil,
		Env:                 "standard",
		EnvVariables:        p.config.EnvVars,
		ErrorHandlers:       nil,
		Handlers: []*appengine.UrlMap{
			{
				Script:        &appengine.ScriptHandler{ScriptPath: "auto"},
				SecurityLevel: "SECURE_ALWAYS",
				UrlRegex:      "/.*",
			},
		},
		HealthCheck:               nil,
		Id:                        versionID,
		InboundServices:           nil,
		InstanceClass:             p.config.InstanceClass,
		Libraries:                 nil,
		LivenessCheck:             nil,
		ManualScaling:             nil,
		Network:                   nil,
		NobuildFilesRegex:         "",
		ReadinessCheck:            nil,
		Resources:                 nil,
		Runtime:                   p.config.Runtime,
		RuntimeApiVersion:         "",
		RuntimeChannel:            "",
		RuntimeMainExecutablePath: p.config.RuntimeMainExecutablePath,
		ServingStatus:             "STOPPED",
		Threadsafe:                false,
		Vm:                        false,
		VpcAccessConnector:        nil,
		ForceSendFields:           nil,
		NullFields:                nil,
	}

	createCall := appengineService.Apps.Services.Versions.Create(appID, service, &aev)
	createCall = createCall.Context(ctx)
	op, err := createCall.Do()
	if err != nil {
		u.Step(terminal.StatusError, "Error creating new App Engine service version")
		return nil, err
	}

	_ = op

	u.Step(terminal.StatusOK, "New service version created '"+versionID+"'")

	return &Deployment{}, nil
}

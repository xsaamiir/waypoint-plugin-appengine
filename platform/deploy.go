package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/sharkyze/waypoint-plugin-cloudstorage/registry"
	"google.golang.org/api/appengine/v1"

	"github.com/sharkyze/waypoint-plugin-appengine/internal/appengineutil"
)

type DeployConfig struct {
	Project string `hcl:"project"`
	Service string `hcl:"service"`
	// Runtime: Desired runtime. Example: go114.
	Runtime string `hcl:"runtime"`
	// InstanceClass: Instance class that is used to run this version. Valid
	// values are: AutomaticScaling: F1, F2, F4, F4_1G ManualScaling or
	// BasicScaling: B1, B2, B4, B8, B4_1GDefaults to F1 for
	// AutomaticScaling and B1 for ManualScaling or BasicScaling.
	InstanceClass             string            `hcl:"instance_class,optional"`
	EnvVars                   map[string]string `hcl:"env_variables,optional"`
	RuntimeMainExecutablePath string            `hcl:"main,optional"`
	AutomaticScaling          *automaticScaling `hcl:"automatic_scaling,block"`
	Handlers                  handlers          `hcl:"handlers,block"`
}

type handler struct {
	URL         string            `hcl:"url"`
	Script      string            `hcl:"script,optional"`
	Secure      string            `hcl:"secure,optional"`
	HTTPHeaders map[string]string `hcl:"http_headers,optional"`
	StaticFiles string            `hcl:"static_files,optional"`
	Upload      string            `hcl:"upload,optional"`
}

type handlers []handler

// toAE converts data to the format expected by the appengine client.
func (h handlers) toAE() []*appengine.UrlMap {
	ums := make([]*appengine.UrlMap, len(h))

	for i, handler := range h {
		var script *appengine.ScriptHandler
		if s := handler.Script; s != "" {
			script = &appengine.ScriptHandler{ScriptPath: s}
		}

		var sfs *appengine.StaticFilesHandler
		if handler.StaticFiles != "" || handler.Upload != "" {
			sfs = &appengine.StaticFilesHandler{
				ApplicationReadable: false,
				Expiration:          "",
				HttpHeaders:         handler.HTTPHeaders,
				MimeType:            "",
				Path:                handler.StaticFiles,
				RequireMatchingFile: false,
				UploadPathRegex:     handler.Upload,
			}
		}

		ums[i] = &appengine.UrlMap{
			ApiEndpoint:              nil,
			AuthFailAction:           "",
			Login:                    "",
			RedirectHttpResponseCode: "",
			Script:                   script,
			SecurityLevel:            handler.Secure,
			StaticFiles:              sfs,
			UrlRegex:                 handler.URL,
		}
	}

	return ums
}

type automaticScaling struct {
	// MaxConcurrentRequests: Number of concurrent requests an automatic
	// scaling instance can accept before the scheduler spawns a new
	// instance.Defaults to a runtime-specific value.
	MaxConcurrentRequests int64 `hcl:"max_concurrent_requests,optional"`

	// MaxIdleInstances: Maximum number of idle instances that should be
	// maintained for this version.
	MaxIdleInstances int64 `hcl:"max_idle_instances,optional"`

	// MaxPendingLatency: Maximum amount of time that a request should wait
	// in the pending queue before starting a new instance to handle it.
	MaxPendingLatency string `hcl:"max_pending_latency,optional"`

	// MaxInstances: Maximum number of instances to run for this version.
	// Set to zero to disable max_instances configuration.
	MaxInstances int64 `hcl:"max_instances,optional"`

	// MinIdleInstances: Minimum number of idle instances that should be
	// maintained for this version. Only applicable for the default version
	// of a service.
	MinIdleInstances int64 `hcl:"min_idle_instances,optional"`

	// MinPendingLatency: Minimum amount of time a request should wait in
	// the pending queue before starting a new instance to handle it.
	MinPendingLatency string `hcl:"min_pending_latency,optional"`

	// MinInstances: Minimum number of instances to run for this version.
	// Set to zero to disable min_instances configuration.
	MinInstances int64 `json:"min_instances,omitempty"`
}

// toAE converts data to the format expected by the appengine client.
func (a *automaticScaling) toAE() *appengine.AutomaticScaling {
	if a == nil {
		return nil
	}

	return &appengine.AutomaticScaling{
		MaxConcurrentRequests: a.MaxConcurrentRequests,
		MaxIdleInstances:      a.MaxIdleInstances,
		MaxPendingLatency:     a.MaxPendingLatency,
		MinIdleInstances:      a.MinIdleInstances,
		MinPendingLatency:     a.MinPendingLatency,
		StandardSchedulerSettings: &appengine.StandardSchedulerSettings{
			MaxInstances: a.MaxInstances,
			MinInstances: a.MinInstances,
		},
	}
}

type Handlers struct {
	URL    string `hcl:"url,optional"`
	Script string `hcl:"script,optional"`
	Secure string `hcl:"secure,optional"`
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
func (p *Platform) deploy(
	ctx context.Context,
	artifact *registry.Artifact,
	ui terminal.UI,
) (*Deployment, error) {
	st := ui.Status()
	defer st.Close()

	st.Update("Creating new App Engine version '" + artifact.Source + "'")

	appengineService, err := appengine.NewService(ctx)
	if err != nil {
		return nil, err
	}

	service := p.config.Service
	project := p.config.Project
	versionID := time.Now().Format("20060102t150405")
	sourceURL := artifact.Source

	aev := appengine.Version{
		ApiConfig:                 nil,
		AutomaticScaling:          p.config.AutomaticScaling.toAE(),
		BasicScaling:              nil,
		BetaSettings:              nil,
		BuildEnvVariables:         nil,
		DefaultExpiration:         "",
		Deployment:                &appengine.Deployment{Zip: &appengine.ZipInfo{SourceUrl: sourceURL}},
		EndpointsApiService:       nil,
		Entrypoint:                &appengine.Entrypoint{Shell: "", ForceSendFields: []string{"Shell"}},
		Env:                       "standard",
		EnvVariables:              p.config.EnvVars,
		ErrorHandlers:             nil,
		Handlers:                  p.config.Handlers.toAE(),
		HealthCheck:               nil,
		Id:                        versionID,
		InboundServices:           nil,
		InstanceClass:             p.config.InstanceClass,
		Libraries:                 nil,
		LivenessCheck:             nil,
		ManualScaling:             nil,
		NobuildFilesRegex:         "",
		ReadinessCheck:            nil,
		Runtime:                   p.config.Runtime,
		RuntimeApiVersion:         "",
		RuntimeChannel:            "",
		RuntimeMainExecutablePath: p.config.RuntimeMainExecutablePath,
		ServingStatus:             "STOPPED",
		Threadsafe:                true,
		Vm:                        false,
		VpcAccessConnector:        nil,
	}

	createCall := appengineService.Apps.Services.Versions.Create(project, service, &aev)
	createCall = createCall.Context(ctx)

	op, err := createCall.Do()
	if err != nil {
		st.Step(terminal.StatusError, "Error creating new App Engine service version")
		return nil, err
	}

	st.Step(terminal.StatusOK, "App Engine version created '"+versionID+"'")
	st.Update("Building new version on Cloud Build '" + op.Name + "'")

	op, err = appengineutil.WaitForOperation(ctx, appengineService, op)
	if err != nil {
		st.Step(terminal.StatusError, "Error fetching the version build status")
		return nil, err
	}

	if op.Error != nil {
		st.Step(terminal.StatusError, "Build error")
		return nil, errors.New(op.Error.Message)
	}

	st.Step(terminal.StatusOK, "New service version created '"+versionID+"'")

	return &Deployment{VersionId: versionID, Project: project, Service: service}, nil
}

package gae

import (
	"context"
	"strings"
	"time"

	"google.golang.org/api/appengine/v1"
)

// operationID parses the operation id out of an operation name.
func operationID(opName string) string {
	// The operation opName has the format: apps/project-id/operations/op-id
	split := strings.Split(opName, "/")
	return split[len(split)-1]
}

// projectID parses the project id out of an operation name.
func projectID(opName string) string {
	// The operation name has the format: apps/project-id/operations/op-id
	split := strings.Split(opName, "/")
	return split[1]
}

// WaitForOperation keeps polling long the operation until it finishes either
// successfully or with an error.
func WaitForOperation(
	ctx context.Context,
	service *appengine.APIService,
	op *appengine.Operation,
) (*appengine.Operation, error) {
	opID := operationID(op.Name)
	app := projectID(op.Name)

	var err error

	for !op.Done {
		opCall := service.Apps.Operations.Get(app, opID)
		opCall = opCall.Context(ctx)
		op, err = opCall.Do()
		if err != nil {
			return nil, err
		}

		time.Sleep(2 * time.Second)
	}

	return op, nil
}

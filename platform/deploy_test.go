package platform

import (
	"testing"
)

func Test_operationID(t *testing.T) {
	type args struct {
		opName string
	}

	tests := []struct {
		args args
		want string
	}{
		{
			args: args{opName: "apps/project-id/operations/op-id"},
			want: "op-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.args.opName, func(t *testing.T) {
			if got := operationID(tt.args.opName); got != tt.want {
				t.Errorf("operationID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_projectID(t *testing.T) {
	type args struct {
		opName string
	}

	tests := []struct {
		args args
		want string
	}{
		{
			args: args{opName: "apps/project-id/operations/op-id"},
			want: "project-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.args.opName, func(t *testing.T) {
			if got := projectID(tt.args.opName); got != tt.want {
				t.Errorf("projectID() = %v, want %v", got, tt.want)
			}
		})
	}
}

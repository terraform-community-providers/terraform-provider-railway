package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Railway switches a service to the Dockerfile builder on any dockerfilePath,
// even "", so an unset dockerfile_path must not be sent at all.
func TestBuildServiceInstanceInputDockerfilePath(t *testing.T) {
	unset := buildServiceInstanceInput(&ServiceResourceModel{}, nil)

	if unset.DockerfilePath != nil {
		t.Fatalf("unset dockerfile_path sent as %q, want omitted", *unset.DockerfilePath)
	}

	set := buildServiceInstanceInput(&ServiceResourceModel{DockerfilePath: types.StringValue("Dockerfile.prod")}, nil)

	if set.DockerfilePath == nil || *set.DockerfilePath != "Dockerfile.prod" {
		t.Fatalf("dockerfile_path not sent: %v", set.DockerfilePath)
	}
}

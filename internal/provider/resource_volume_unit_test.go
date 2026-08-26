package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVolumeCreateInputDetachedWireFormat(t *testing.T) {
	client := &recordingGraphQLClient{}
	_, err := createManagedVolume(
		context.Background(),
		client,
		"project-id",
		detachedVolumeMountPath,
	)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	if !strings.Contains(client.request.Query, "serviceId:null") {
		t.Fatalf("create request does not explicitly detach the volume:\n%s", client.request.Query)
	}
	if !strings.Contains(client.request.Query, "environmentId:null") {
		t.Fatalf("create request does not explicitly create zero environment instances:\n%s", client.request.Query)
	}
}

type recordingGraphQLClient struct {
	request *graphql.Request
}

func (c *recordingGraphQLClient) MakeRequest(
	ctx context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	c.request = req
	return nil
}

func TestVolumeInstanceDetachInputWireFormat(t *testing.T) {
	input := __updateManagedVolumeInstanceInput{
		VolumeId:      "volume-id",
		EnvironmentId: "environment-id",
		ServiceId:     nil,
		MountPath:     "/data",
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshalling input: %v", err)
	}

	const expected = `{"volumeId":"volume-id","environmentId":"environment-id","serviceId":null,"mountPath":"/data"}`
	if string(encoded) != expected {
		t.Fatalf("unexpected wire format:\nwant: %s\n got: %s", expected, encoded)
	}
}

func TestParseVolumeImportId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		first         string
		second        string
		shouldSucceed bool
	}{
		{name: "volume", id: "project-id:volume-id", first: "project-id", second: "volume-id", shouldSucceed: true},
		{name: "missing volume", id: "project-id:", shouldSucceed: false},
		{name: "missing project", id: ":volume-id", shouldSucceed: false},
		{name: "extra component", id: "project-id:volume-id:extra", shouldSucceed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			first, second, ok := parseVolumeImportId(test.id)
			if ok != test.shouldSucceed || first != test.first || second != test.second {
				t.Fatalf("parseVolumeImportId(%q) = (%q, %q, %t)", test.id, first, second, ok)
			}
		})
	}
}

func TestApplyManagedVolumeInstanceReflectsExternalDetach(t *testing.T) {
	t.Parallel()

	state := VolumeInstanceResourceModel{
		ServiceId: types.StringValue("previous-service-id"),
	}
	instance := ManagedVolumeInstance{
		Id:            "instance-id",
		VolumeId:      "volume-id",
		EnvironmentId: "environment-id",
		ServiceId:     nil,
		MountPath:     "/external-path",
		SizeMB:        5000,
	}

	applyManagedVolumeInstance(&state, instance)

	if !state.ServiceId.IsNull() {
		t.Fatalf("expected external detach to set service_id to null, got %q", state.ServiceId.ValueString())
	}
	if !state.MountPath.IsNull() {
		t.Fatalf("expected external detach to clear mount_path, got %q", state.MountPath.ValueString())
	}
	if state.SizeMB.ValueInt64() != 5000 {
		t.Fatalf("expected size_mb 5000, got %d", state.SizeMB.ValueInt64())
	}
}

package provider

import (
	"context"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

type environmentPatchClient struct {
	makeRequest func(context.Context, *graphql.Request, *graphql.Response) error
}

func (c environmentPatchClient) MakeRequest(
	ctx context.Context,
	request *graphql.Request,
	response *graphql.Response,
) error {
	return c.makeRequest(ctx, request, response)
}

func TestCommitAndWaitForEnvironmentPatch(t *testing.T) {
	t.Parallel()

	client := environmentPatchClient{
		makeRequest: func(_ context.Context, request *graphql.Request, response *graphql.Response) error {
			switch request.OpName {
			case "getEnvironmentPatchLifecycle":
				return nil
			case "commitEnvironmentPatch":
				response.Data.(*commitEnvironmentPatchResponse).EnvironmentPatchCommit =
					"https://backboard.railway.com/environment-patches/patch-id"
				return nil
			case "getEnvironmentPatch":
				if request.Variables.(*__getEnvironmentPatchInput).Id != "patch-id" {
					t.Fatalf("expected normalized patch ID")
				}
				patch := &response.Data.(*getEnvironmentPatchResponse).EnvironmentPatch
				patch.Status = EnvironmentPatchStatusCommitted
				patch.LastAppliedError = "superseded apply attempt"
				return nil
			default:
				t.Fatalf("unexpected operation %q", request.OpName)
				return nil
			}
		},
	}

	err := commitAndWaitForEnvironmentPatch(
		context.Background(),
		client,
		"environment-id",
		railway.EnvironmentConfig{},
		"Test patch",
	)
	if err != nil {
		t.Fatalf("expected patch to commit: %v", err)
	}
}

func TestCommitEnvironmentPatchRejectsPendingStagedChanges(t *testing.T) {
	t.Parallel()

	client := environmentPatchClient{
		makeRequest: func(_ context.Context, request *graphql.Request, response *graphql.Response) error {
			if request.OpName != "getEnvironmentPatchLifecycle" {
				t.Fatalf("unexpected operation %q", request.OpName)
			}
			response.Data.(*getEnvironmentPatchLifecycleResponse).
				Environment.UnmergedChangesCount = 1
			return nil
		},
	}

	err := commitAndWaitForEnvironmentPatch(
		context.Background(),
		client,
		"environment-id",
		railway.EnvironmentConfig{},
		"Test patch",
	)
	if err == nil ||
		err.Error() != "Railway environment has pending staged changes; commit or discard them before applying Terraform configuration" {
		t.Fatalf("expected pending staged changes error, got %v", err)
	}
}

func TestWaitForEnvironmentPatchReturnsApplyError(t *testing.T) {
	t.Parallel()

	client := environmentPatchClient{
		makeRequest: func(_ context.Context, _ *graphql.Request, response *graphql.Response) error {
			patch := &response.Data.(*getEnvironmentPatchResponse).EnvironmentPatch
			patch.Status = EnvironmentPatchStatusApplying
			patch.LastAppliedError = "invalid configuration"
			return nil
		},
	}

	err := waitForEnvironmentPatch(context.Background(), client, "patch-id")
	if err == nil || err.Error() != "Railway failed to apply environment patch: invalid configuration" {
		t.Fatalf("expected Railway apply error, got %v", err)
	}
}

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/terraform-community-providers/terraform-provider-railway/internal/railway"
)

const (
	environmentPatchPollInterval = time.Second
	environmentPatchTimeout      = 30 * time.Second
)

func commitAndWaitForEnvironmentPatch(
	ctx context.Context,
	client graphql.Client,
	environmentID string,
	patch railway.EnvironmentConfig,
	commitMessage string,
) error {
	state, err := getEnvironmentPatchLifecycle(ctx, client, environmentID)
	if err != nil {
		return err
	}
	if state.Environment.UnmergedChangesCount > 0 {
		return fmt.Errorf(
			"Railway environment has pending staged changes; commit or discard them before applying Terraform configuration",
		)
	}

	response, err := commitEnvironmentPatch(ctx, client, environmentID, patch, commitMessage)
	if err != nil {
		return err
	}
	return waitForEnvironmentPatch(ctx, client, response.EnvironmentPatchCommit)
}

func waitForEnvironmentPatch(ctx context.Context, client graphql.Client, patchID string) error {
	patchID = environmentPatchID(patchID)
	ticker := time.NewTicker(environmentPatchPollInterval)
	defer ticker.Stop()
	timer := time.NewTimer(environmentPatchTimeout)
	defer timer.Stop()

	for {
		response, err := getEnvironmentPatch(ctx, client, patchID)
		if err != nil {
			return err
		}
		patch := response.EnvironmentPatch
		if patch.Status == EnvironmentPatchStatusCommitted {
			return nil
		}
		if patch.LastAppliedError != "" {
			return fmt.Errorf("Railway failed to apply environment patch: %s", patch.LastAppliedError)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timed out waiting for Railway environment patch")
		case <-ticker.C:
		}
	}
}

func environmentPatchID(patchID string) string {
	if separator := strings.LastIndexByte(patchID, '/'); separator >= 0 {
		return patchID[separator+1:]
	}
	return patchID
}

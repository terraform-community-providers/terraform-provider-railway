//go:build tools

package tools

import (
	// GraphQL client generation
	_ "github.com/Khan/genqlient/generate"

	// Documentation generation
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

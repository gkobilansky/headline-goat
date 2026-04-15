package deploy

import (
	"context"
	"fmt"
	"strings"
)

// installInstruction returns a short, actionable hint for each provider's
// CLI (what to install and how to authenticate). Keyed by Provider.Name().
var installInstruction = map[string]string{
	"do":      "install doctl and run `doctl auth init` (https://docs.digitalocean.com/reference/doctl/how-to/install/)",
	"hetzner": "install hcloud and run `hcloud context create <name>` (https://github.com/hetznercloud/cli)",
}

// SelectProvider chooses a Provider based on an optional explicit name and
// the runtime availability of each candidate's CLI.
//
// Rules:
//   - explicit name set → must exist AND be authenticated; else clear error
//   - no explicit name AND exactly one available → that one
//   - no explicit name AND multiple available → error asking for --provider
//   - no explicit name AND none available → error listing install instructions
func SelectProvider(ctx context.Context, explicit string, providers []Provider) (Provider, error) {
	if explicit != "" {
		for _, p := range providers {
			if p.Name() == explicit {
				if !p.Available(ctx) {
					return nil, fmt.Errorf("provider %q is not authenticated — %s", explicit, installInstruction[p.Name()])
				}
				return p, nil
			}
		}
		return nil, fmt.Errorf("unknown provider %q (supported: %s)", explicit, strings.Join(providerNames(providers), ", "))
	}

	var available []Provider
	for _, p := range providers {
		if p.Available(ctx) {
			available = append(available, p)
		}
	}

	switch len(available) {
	case 1:
		return available[0], nil
	case 0:
		var hints []string
		for _, p := range providers {
			hints = append(hints, "  - "+installInstruction[p.Name()])
		}
		return nil, fmt.Errorf("no authenticated provider CLI found. Install one of:\n%s", strings.Join(hints, "\n"))
	default:
		names := make([]string, 0, len(available))
		for _, p := range available {
			names = append(names, p.Name())
		}
		return nil, fmt.Errorf("multiple authenticated providers (%s); pass --provider to choose one", strings.Join(names, ", "))
	}
}

func providerNames(ps []Provider) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name()
	}
	return out
}

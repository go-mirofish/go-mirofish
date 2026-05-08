package wasm

import (
	"context"
	"fmt"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

// LoadFromDir discovers a manifest+module pair in a directory, compiles the guest,
// and validates it against the manifest-derived contract.
func (r *Runtime) LoadFromDir(ctx context.Context, dir string) (*Compiled, plugins.Manifest, error) {
	reg, err := plugins.DiscoverDirectory(dir)
	if err != nil {
		return nil, plugins.Manifest{}, err
	}
	if reg.Manifest.Runtime != "wasm" {
		return nil, plugins.Manifest{}, fmt.Errorf("plugin runtime mismatch: got %q", reg.Manifest.Runtime)
	}
	compiled, err := r.CompileFile(ctx, reg.ModulePath)
	if err != nil {
		return nil, plugins.Manifest{}, err
	}
	contract := ContractFromManifest(reg.Manifest)
	if err := compiled.Validate(ctx, contract); err != nil {
		return nil, plugins.Manifest{}, err
	}
	return compiled, reg.Manifest, nil
}

package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const DefaultManifestFilename = "manifest.json"

// Registration describes a discovered plugin directory.
type Registration struct {
	Directory    string   `json:"directory"`
	ManifestPath string   `json:"manifest_path"`
	ModulePath   string   `json:"module_path"`
	Manifest     Manifest `json:"manifest"`
}

// Registry tracks discovered plugins by name.
type Registry struct {
	mu            sync.RWMutex
	registrations map[string]Registration
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{registrations: map[string]Registration{}}
}

// RegisterDirectory loads a manifest from the given directory and registers it.
func (r *Registry) RegisterDirectory(dir string) (Registration, error) {
	if r == nil {
		return Registration{}, fmt.Errorf("plugin registry is nil")
	}
	reg, err := DiscoverDirectory(dir)
	if err != nil {
		return Registration{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registrations[reg.Manifest.Name] = reg
	return reg, nil
}

// Get returns a registration by plugin name.
func (r *Registry) Get(name string) (Registration, bool) {
	if r == nil {
		return Registration{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.registrations[name]
	return reg, ok
}

// List returns the registered plugins sorted by name.
func (r *Registry) List() []Registration {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Registration, 0, len(r.registrations))
	for _, reg := range r.registrations {
		out = append(out, reg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest.Name < out[j].Manifest.Name
	})
	return out
}

// DiscoverDirectory resolves a plugin directory into a normalized registration.
func DiscoverDirectory(dir string) (Registration, error) {
	dir = filepath.Clean(dir)
	manifestPath := filepath.Join(dir, DefaultManifestFilename)
	manifest, err := LoadManifestFile(manifestPath)
	if err != nil {
		return Registration{}, err
	}
	moduleName := strings.TrimSpace(manifest.Module)
	if moduleName == "" {
		moduleName, err = DefaultModuleForRuntime(manifest.Runtime)
		if err != nil {
			return Registration{}, err
		}
	}
	modulePath := filepath.Clean(filepath.Join(dir, moduleName))
	rel, err := filepath.Rel(dir, modulePath)
	if err != nil {
		return Registration{}, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Registration{}, fmt.Errorf("plugin module path escapes plugin directory: %q", moduleName)
	}
	if _, err := os.Stat(modulePath); err != nil {
		return Registration{}, err
	}
	return Registration{
		Directory:    dir,
		ManifestPath: manifestPath,
		ModulePath:   modulePath,
		Manifest: Manifest{
			Name:         manifest.Name,
			Version:      manifest.Version,
			Runtime:      manifest.Runtime,
			APIVersion:   manifest.APIVersion,
			Module:       moduleName,
			Entry:        manifest.Entry,
			Allocate:     manifest.Allocate,
			Deallocate:   manifest.Deallocate,
			ABIFunction:  manifest.ABIFunction,
			DigestSHA256: manifest.DigestSHA256,
			SignerID:     manifest.SignerID,
			Signature:    manifest.Signature,
			Capabilities: append([]string(nil), manifest.Capabilities...),
			Start:        append([]string(nil), manifest.Start...),
		},
	}, nil
}

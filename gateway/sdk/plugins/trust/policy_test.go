package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

func TestVerifyManifestAndModule(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	module := []byte("plugin-module")
	digest := sha256.Sum256(module)
	manifest := plugins.Manifest{
		Name:         "trusted-plugin",
		Version:      "0.1.0",
		Runtime:      "wasm",
		APIVersion:   1,
		Entry:        "run",
		DigestSHA256: hex.EncodeToString(digest[:]),
		SignerID:     "core",
	}
	payload, err := SigningPayload(manifest)
	if err != nil {
		t.Fatalf("signingPayload: %v", err)
	}
	manifest.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))
	policy := Policy{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]ed25519.PublicKey{
			"core": pub,
		},
	}
	if err := VerifyManifestAndModule(policy, manifest, module); err != nil {
		t.Fatalf("VerifyManifestAndModule: %v", err)
	}
}

func TestVerifyManifestAndModuleRejectsBadDigest(t *testing.T) {
	manifest := plugins.Manifest{
		Name:         "bad-plugin",
		Version:      "0.1.0",
		Runtime:      "wasm",
		APIVersion:   1,
		Entry:        "run",
		DigestSHA256: "deadbeef",
	}
	err := VerifyManifestAndModule(Policy{RequireDigest: true}, manifest, []byte("plugin-module"))
	if err == nil {
		t.Fatal("expected digest error")
	}
}

func TestParsePolicy(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	raw, err := json.Marshal(Config{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]string{
			"core": base64.StdEncoding.EncodeToString(pub),
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	policy, err := ParsePolicy(raw)
	if err != nil {
		t.Fatalf("ParsePolicy: %v", err)
	}
	if !policy.RequireDigest || !policy.RequireSigned {
		t.Fatalf("expected strict policy, got %#v", policy)
	}
	if len(policy.TrustedSigners["core"]) != ed25519.PublicKeySize {
		t.Fatalf("unexpected signer key length: %d", len(policy.TrustedSigners["core"]))
	}
}

func TestLoadPolicyFile(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "trust.json")
	raw, err := json.Marshal(Config{
		RequireDigest: true,
		TrustedSigners: map[string]string{
			"core": hex.EncodeToString(pub),
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	policy, err := LoadPolicyFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyFile: %v", err)
	}
	if !policy.RequireDigest {
		t.Fatal("expected require_digest=true")
	}
	if len(policy.TrustedSigners["core"]) != ed25519.PublicKeySize {
		t.Fatalf("unexpected signer key length: %d", len(policy.TrustedSigners["core"]))
	}
}

func TestParsePolicyRejectsBadSignerKey(t *testing.T) {
	raw := []byte(`{
		"trusted_signers": {
			"core": "not-a-valid-key"
		}
	}`)
	if _, err := ParsePolicy(raw); err == nil {
		t.Fatal("expected signer decode error")
	}
}

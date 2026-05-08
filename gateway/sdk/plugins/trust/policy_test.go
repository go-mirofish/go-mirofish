package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

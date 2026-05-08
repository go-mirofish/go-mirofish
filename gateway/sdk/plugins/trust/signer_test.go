package trust

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

func TestSignManifestAndModule(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	module := []byte("plugin-module")
	manifest := plugins.Manifest{
		Name:       "signed-plugin",
		Version:    "0.1.0",
		Runtime:    plugins.RuntimeWasm,
		APIVersion: 1,
		Entry:      "run",
	}
	signed, err := SignManifestAndModule(manifest, module, "core", priv)
	if err != nil {
		t.Fatalf("SignManifestAndModule: %v", err)
	}
	if signed.DigestSHA256 == "" || signed.Signature == "" {
		t.Fatalf("expected digest and signature, got %#v", signed)
	}
	err = VerifyManifestAndModule(Policy{
		RequireDigest: true,
		RequireSigned: true,
		TrustedSigners: map[string]ed25519.PublicKey{
			"core": pub,
		},
	}, signed, module)
	if err != nil {
		t.Fatalf("VerifyManifestAndModule: %v", err)
	}
}

func TestParsePrivateKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	for _, raw := range [][]byte{
		[]byte(base64.StdEncoding.EncodeToString(priv)),
		[]byte(hex.EncodeToString(priv)),
		[]byte(base64.StdEncoding.EncodeToString(priv.Seed())),
		[]byte(hex.EncodeToString(priv.Seed())),
	} {
		key, err := ParsePrivateKey(raw)
		if err != nil {
			t.Fatalf("ParsePrivateKey: %v", err)
		}
		if len(key) != ed25519.PrivateKeySize {
			t.Fatalf("unexpected private key size: %d", len(key))
		}
	}
}

func TestLoadPrivateKeyFile(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	path := filepath.Join(t.TempDir(), "signing.key")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(priv.Seed())), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	key, err := LoadPrivateKeyFile(path)
	if err != nil {
		t.Fatalf("LoadPrivateKeyFile: %v", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		t.Fatalf("unexpected private key size: %d", len(key))
	}
}

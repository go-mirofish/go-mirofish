package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

// SignManifestAndModule binds a manifest to module bytes and returns a signed copy.
func SignManifestAndModule(manifest plugins.Manifest, module []byte, signerID string, privateKey ed25519.PrivateKey) (plugins.Manifest, error) {
	manifest.Runtime = plugins.NormalizeRuntime(manifest.Runtime)
	if strings.TrimSpace(manifest.Module) == "" {
		if moduleName, err := plugins.DefaultModuleForRuntime(manifest.Runtime); err == nil {
			manifest.Module = moduleName
		}
	}
	if err := plugins.ValidateManifest(manifest); err != nil {
		return plugins.Manifest{}, err
	}
	if strings.TrimSpace(signerID) == "" {
		return plugins.Manifest{}, fmt.Errorf("plugin trust: signer id is required")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return plugins.Manifest{}, fmt.Errorf("plugin trust: expected %d-byte private key, got %d", ed25519.PrivateKeySize, len(privateKey))
	}
	digest := sha256.Sum256(module)
	signed := manifest
	signed.DigestSHA256 = hex.EncodeToString(digest[:])
	signed.SignerID = strings.TrimSpace(signerID)
	signed.Signature = ""
	payload, err := SigningPayload(signed)
	if err != nil {
		return plugins.Manifest{}, err
	}
	signed.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return signed, nil
}

// ParsePrivateKey decodes a base64 or hex Ed25519 private key or seed.
func ParsePrivateKey(raw []byte) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("plugin trust: missing private key")
	}
	key, err := decodeSigningKey(trimmed)
	if err != nil {
		return nil, err
	}
	switch len(key) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(key), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(key), nil
	default:
		return nil, fmt.Errorf("plugin trust: expected %d-byte private key or %d-byte seed, got %d", ed25519.PrivateKeySize, ed25519.SeedSize, len(key))
	}
}

// LoadPrivateKeyFile reads and parses a private signing key from disk.
func LoadPrivateKeyFile(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePrivateKey(raw)
}

func decodeSigningKey(encoded string) ([]byte, error) {
	if looksLikeHex(encoded) || len(encoded) == ed25519.PrivateKeySize*2 || len(encoded) == ed25519.SeedSize*2 {
		key, err := hex.DecodeString(encoded)
		if err == nil {
			return key, nil
		}
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return key, nil
	}
	key, hexErr := hex.DecodeString(encoded)
	if hexErr == nil {
		return key, nil
	}
	return nil, fmt.Errorf("plugin trust: expected base64 or hex private key")
}

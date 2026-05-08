package trust

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config is the JSON-serializable trust policy surface for SDK plugin loading.
type Config struct {
	RequireDigest  bool              `json:"require_digest"`
	RequireSigned  bool              `json:"require_signed"`
	AllowUnsigned  bool              `json:"allow_unsigned"`
	TrustedSigners map[string]string `json:"trusted_signers"`
}

// ParsePolicy decodes a JSON trust configuration into a runtime policy.
func ParsePolicy(raw []byte) (Policy, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Policy{}, err
	}
	policy := Policy{
		RequireDigest:  cfg.RequireDigest,
		RequireSigned:  cfg.RequireSigned,
		AllowUnsigned:  cfg.AllowUnsigned,
		TrustedSigners: make(map[string]ed25519.PublicKey, len(cfg.TrustedSigners)),
	}
	for signerID, encodedKey := range cfg.TrustedSigners {
		signerID = strings.TrimSpace(signerID)
		if signerID == "" {
			return Policy{}, fmt.Errorf("plugin trust: trusted_signers contains an empty signer id")
		}
		key, err := parsePublicKey(encodedKey)
		if err != nil {
			return Policy{}, fmt.Errorf("plugin trust: decode signer %q: %w", signerID, err)
		}
		policy.TrustedSigners[signerID] = key
	}
	return policy, nil
}

// LoadPolicyFile reads and parses a trust policy from disk.
func LoadPolicyFile(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	return ParsePolicy(raw)
}

func parsePublicKey(encoded string) (ed25519.PublicKey, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, fmt.Errorf("missing public key")
	}
	var (
		key []byte
		err error
	)
	if looksLikeHex(encoded) {
		key, err = hex.DecodeString(encoded)
	} else {
		key, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			key, err = hex.DecodeString(encoded)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("expected base64 or hex public key")
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected %d-byte public key, got %d", ed25519.PublicKeySize, len(key))
	}
	return ed25519.PublicKey(key), nil
}

func looksLikeHex(s string) bool {
	if len(s) != ed25519.PublicKeySize*2 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

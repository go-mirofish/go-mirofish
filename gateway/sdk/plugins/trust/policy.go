package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

type Policy struct {
	RequireDigest  bool
	RequireSigned  bool
	AllowUnsigned  bool
	TrustedSigners map[string]ed25519.PublicKey
}

func VerifyManifestAndModule(policy Policy, manifest plugins.Manifest, module []byte) error {
	digest := sha256.Sum256(module)
	digestHex := hex.EncodeToString(digest[:])

	if policy.RequireDigest || manifest.DigestSHA256 != "" || manifest.Signature != "" {
		if manifest.DigestSHA256 == "" {
			return errors.New("plugin trust: missing digest_sha256")
		}
		if manifest.DigestSHA256 != digestHex {
			return fmt.Errorf("plugin trust: digest mismatch got %s want %s", digestHex, manifest.DigestSHA256)
		}
	}

	if manifest.Signature == "" {
		if policy.RequireSigned && !policy.AllowUnsigned {
			return errors.New("plugin trust: signature required")
		}
		return nil
	}

	if manifest.SignerID == "" {
		return errors.New("plugin trust: signature present but signer_id missing")
	}
	key, ok := policy.TrustedSigners[manifest.SignerID]
	if !ok {
		return fmt.Errorf("plugin trust: untrusted signer %q", manifest.SignerID)
	}

	payload, err := SigningPayload(manifest)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		return fmt.Errorf("plugin trust: decode signature: %w", err)
	}
	if !ed25519.Verify(key, payload, sig) {
		return errors.New("plugin trust: signature verification failed")
	}
	return nil
}

func SigningPayload(manifest plugins.Manifest) ([]byte, error) {
	caps := append([]string(nil), manifest.Capabilities...)
	sort.Strings(caps)
	start := append([]string(nil), manifest.Start...)
	sort.Strings(start)
	payload := struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Runtime      string   `json:"runtime"`
		APIVersion   uint64   `json:"api_version"`
		Module       string   `json:"module,omitempty"`
		Entry        string   `json:"entry"`
		Allocate     string   `json:"allocate,omitempty"`
		Deallocate   string   `json:"deallocate,omitempty"`
		ABIFunction  string   `json:"abi_function,omitempty"`
		DigestSHA256 string   `json:"digest_sha256,omitempty"`
		Capabilities []string `json:"capabilities,omitempty"`
		Start        []string `json:"start,omitempty"`
	}{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Runtime:      manifest.Runtime,
		APIVersion:   manifest.APIVersion,
		Module:       manifest.Module,
		Entry:        manifest.Entry,
		Allocate:     manifest.Allocate,
		Deallocate:   manifest.Deallocate,
		ABIFunction:  manifest.ABIFunction,
		DigestSHA256: manifest.DigestSHA256,
		Capabilities: caps,
		Start:        start,
	}
	return json.Marshal(payload)
}

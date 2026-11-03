// Package identity gives a device a long-lived self-signed certificate. The
// certificate's fingerprint identifies the device, so other devices can pin it
// without any certificate authority.
package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	directoryName = "DeckDrop"
	certFileName  = "identity.crt"
	keyFileName   = "identity.key"
	validFor      = 20 * 365 * 24 * time.Hour
)

// Identity is a device's certificate and private key.
type Identity struct {
	certificate tls.Certificate
	fingerprint string
}

// DefaultDir is where the identity lives for the current user.
func DefaultDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(configDir, directoryName), nil
}

// Generate creates a new self-signed identity that is not saved anywhere.
func Generate() (*Identity, error) {
	certPEM, keyPEM, err := generatePEM()
	if err != nil {
		return nil, err
	}
	return fromPEM(certPEM, keyPEM)
}

// LoadOrCreate returns the identity saved in dir, creating and saving one
// first if there is none. The private key is readable only by the user.
func LoadOrCreate(dir string) (*Identity, error) {
	certPath := filepath.Join(dir, certFileName)
	keyPath := filepath.Join(dir, keyFileName)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		identity, err := fromPEM(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("load identity from %s: %w", dir, err)
		}
		return identity, nil
	}
	for _, err := range []error{certErr, keyErr} {
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read identity: %w", err)
		}
	}

	certPEM, keyPEM, err := generatePEM()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create identity directory: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("save identity key: %w", err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, fmt.Errorf("save identity certificate: %w", err)
	}
	return fromPEM(certPEM, keyPEM)
}

// Fingerprint is the hex SHA-256 of the certificate.
func (identity *Identity) Fingerprint() string {
	return identity.fingerprint
}

// ServerConfig is the TLS configuration for serving as this device.
func (identity *Identity) ServerConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{identity.certificate},
		MinVersion:   tls.VersionTLS13,
	}
}

// ClientConfig is the TLS configuration for connecting to another device.
// Self-signed certificates cannot be checked against a certificate authority,
// so the connection is accepted only if the device's certificate has the
// pinned fingerprint. An empty pin accepts any certificate: the connection is
// encrypted but the device is not authenticated.
func ClientConfig(pin string) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // #nosec G402 -- the certificate is verified by VerifyPeerCertificate
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("device presented no certificate")
			}
			if pin != "" && Fingerprint(rawCerts[0]) != pin {
				return ErrFingerprintMismatch
			}
			return nil
		},
	}
}

// ErrFingerprintMismatch means a device's certificate is not the pinned one.
var ErrFingerprintMismatch = errors.New("device certificate does not match the pinned fingerprint")

// Fingerprint returns the fingerprint of a DER encoded certificate.
func Fingerprint(certificate []byte) string {
	sum := sha256.Sum256(certificate)
	return hex.EncodeToString(sum[:])
}

func generatePEM() (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial number: %w", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "DeckDrop"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("encode key: %w", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func fromPEM(certPEM, keyPEM []byte) (*Identity, error) {
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse certificate and key: %w", err)
	}
	return &Identity{certificate: certificate, fingerprint: Fingerprint(certificate.Certificate[0])}, nil
}

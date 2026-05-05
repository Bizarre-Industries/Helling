package incus

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	userCertificateValidity = 90 * 24 * time.Hour
	internalCAValidity      = 5 * 365 * 24 * time.Hour
)

const (
	// DefaultHellingCAKeyPath is the encrypted Helling CA private key path.
	DefaultHellingCAKeyPath = "/var/lib/helling/ca.key.age"
	// DefaultHellingCACertPath is the public Helling CA certificate path.
	DefaultHellingCACertPath = "/var/lib/helling/ca.crt"
)

// CertificateAuthority signs per-user Incus client certificates.
type CertificateAuthority struct {
	Certificate *x509.Certificate
	PrivateKey  ed25519.PrivateKey
}

// NewInternalCertificateAuthority creates the Helling internal CA certificate
// used to issue per-user Incus client certificates.
func NewInternalCertificateAuthority(generatedAt time.Time) (*CertificateAuthority, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating internal CA key: %w", err)
	}
	serial, err := certificateSerial()
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Helling CA",
			Organization: []string{"Bizarre Industries"},
			Country:      []string{"US"},
		},
		NotBefore:             generatedAt.Add(-5 * time.Minute),
		NotAfter:              generatedAt.Add(internalCAValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("creating internal CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parsing internal CA certificate: %w", err)
	}
	return &CertificateAuthority{Certificate: cert, PrivateKey: privateKey}, nil
}

// IssueUserCertificate creates a per-user Ed25519 client certificate for Incus mTLS.
func IssueUserCertificate(userID int64, username string, issuedAt time.Time) (certPEM, keyPEM []byte, fingerprint string, expiresAt time.Time, err error) {
	ca, err := NewInternalCertificateAuthority(issuedAt)
	if err != nil {
		return nil, nil, "", time.Time{}, err
	}
	return IssueUserCertificateWithCA(userID, username, issuedAt, ca)
}

// IssueUserCertificateWithCA creates a per-user Ed25519 client certificate
// signed by the Helling internal CA.
func IssueUserCertificateWithCA(
	userID int64,
	username string,
	issuedAt time.Time,
	ca *CertificateAuthority,
) (certPEM, keyPEM []byte, fingerprint string, expiresAt time.Time, err error) {
	username = strings.TrimSpace(username)
	if userID <= 0 || username == "" {
		return nil, nil, "", time.Time{}, errors.New("user id and username are required")
	}
	if ca == nil || ca.Certificate == nil || ca.PrivateKey == nil {
		return nil, nil, "", time.Time{}, errors.New("internal CA is required")
	}
	if !ca.Certificate.IsCA {
		return nil, nil, "", time.Time{}, errors.New("internal CA certificate must be a CA")
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, "", time.Time{}, fmt.Errorf("generating user certificate key: %w", err)
	}
	serial, err := certificateSerial()
	if err != nil {
		return nil, nil, "", time.Time{}, err
	}
	expiresAt = issuedAt.Add(userCertificateValidity)
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("helling-user-%d-%s", userID, username),
			Organization: []string{"Helling Users"},
		},
		NotBefore:             issuedAt.Add(-5 * time.Minute),
		NotAfter:              expiresAt,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate, publicKey, ca.PrivateKey)
	if err != nil {
		return nil, nil, "", time.Time{}, fmt.Errorf("creating user certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, "", time.Time{}, fmt.Errorf("marshaling user certificate key: %w", err)
	}
	sum := sha256.Sum256(certDER)
	fingerprint = hex.EncodeToString(sum[:])
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, fingerprint, expiresAt, nil
}

// LoadOrCreateCertificateAuthority loads the persistent Helling CA from disk
// or creates and stores it. The CA private key is encrypted by the caller.
func LoadOrCreateCertificateAuthority(
	keyPath string,
	certPath string,
	encrypt func(string) (string, error),
	decrypt func(string) (string, error),
	generatedAt time.Time,
) (*CertificateAuthority, error) {
	if keyPath == "" || certPath == "" {
		return nil, errors.New("CA key and certificate paths are required")
	}
	stored, err := readStoredCertificateAuthorityFiles(keyPath, certPath)
	if err != nil {
		return nil, err
	}
	if stored.Exists {
		return parseStoredCertificateAuthority(stored.EncryptedKey, stored.CertPEM, decrypt)
	}
	return createAndStoreCertificateAuthority(keyPath, certPath, encrypt, generatedAt)
}

type storedCertificateAuthorityFiles struct {
	EncryptedKey string
	CertPEM      []byte
	Exists       bool
}

func readStoredCertificateAuthorityFiles(keyPath, certPath string) (storedCertificateAuthorityFiles, error) {
	keyCiphertext, keyErr := os.ReadFile(keyPath) // #nosec G304 -- operator-configured CA path.
	certPEM, certErr := os.ReadFile(certPath)     // #nosec G304 -- operator-configured CA path.
	if keyErr == nil && certErr == nil {
		return storedCertificateAuthorityFiles{EncryptedKey: string(keyCiphertext), CertPEM: certPEM, Exists: true}, nil
	}
	keyMissing := errors.Is(keyErr, os.ErrNotExist)
	certMissing := errors.Is(certErr, os.ErrNotExist)
	if keyErr != nil && !keyMissing {
		return storedCertificateAuthorityFiles{}, fmt.Errorf("reading CA key %s: %w", keyPath, keyErr)
	}
	if certErr != nil && !certMissing {
		return storedCertificateAuthorityFiles{}, fmt.Errorf("reading CA certificate %s: %w", certPath, certErr)
	}
	if keyMissing != certMissing {
		return storedCertificateAuthorityFiles{}, fmt.Errorf("CA key and certificate files must be created together: key_missing=%t cert_missing=%t", keyMissing, certMissing)
	}
	return storedCertificateAuthorityFiles{}, nil
}

func createAndStoreCertificateAuthority(
	keyPath string,
	certPath string,
	encrypt func(string) (string, error),
	generatedAt time.Time,
) (*CertificateAuthority, error) {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o750); err != nil {
		return nil, fmt.Errorf("creating CA key directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o750); err != nil {
		return nil, fmt.Errorf("creating CA certificate directory: %w", err)
	}
	ca, err := NewInternalCertificateAuthority(generatedAt)
	if err != nil {
		return nil, err
	}
	keyPEM, err := marshalCAPrivateKey(ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	if encrypt == nil {
		return nil, errors.New("CA key encrypt function is required")
	}
	encryptedKey, err := encrypt(string(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("encrypting CA private key: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(encryptedKey), 0o600); err != nil {
		return nil, fmt.Errorf("writing encrypted CA key: %w", err)
	}
	// #nosec G306 -- the public CA certificate contains no secrets.
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Certificate.Raw}), 0o644); err != nil {
		return nil, fmt.Errorf("writing CA certificate: %w", err)
	}
	return ca, nil
}

func parseStoredCertificateAuthority(encryptedKey string, certPEM []byte, decrypt func(string) (string, error)) (*CertificateAuthority, error) {
	if decrypt == nil {
		return nil, errors.New("CA key decrypt function is required")
	}
	keyPEM, err := decrypt(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting CA private key: %w", err)
	}
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil || keyBlock.Type != "PRIVATE KEY" {
		return nil, errors.New("CA private key PEM is invalid")
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CA private key: %w", err)
	}
	key, ok := keyAny.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("CA private key must be Ed25519")
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, errors.New("CA certificate PEM is invalid")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, errors.New("stored Helling certificate is not a CA")
	}
	return &CertificateAuthority{Certificate: cert, PrivateKey: key}, nil
}

func marshalCAPrivateKey(key ed25519.PrivateKey) ([]byte, error) {
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshaling CA private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}

func certificateSerial() (*big.Int, error) {
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("generating certificate serial: %w", err)
	}
	return serial, nil
}

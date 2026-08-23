package eds

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	pkcs7 "github.com/ddulesov/pkcs7"
)

// genCert генерирует самоподписанный RSA-сертификат с заданным subject.
func genCert(t *testing.T, subj pkix.Name) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      subj,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func TestExtractIdentity(t *testing.T) {
	der := genCert(t, pkix.Name{
		CommonName:         "ИВАНОВ ИВАН",
		SerialNumber:       "IIN990101300123",
		OrganizationalUnit: []string{"BIN123456789012"},
	})
	c, err := pkcs7.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	id, err := extractIdentity(c)
	if err != nil {
		t.Fatalf("extractIdentity: %v", err)
	}
	if id.IIN != "990101300123" {
		t.Errorf("IIN = %q, want 990101300123", id.IIN)
	}
	if id.BIN != "123456789012" {
		t.Errorf("BIN = %q, want 123456789012", id.BIN)
	}
	if id.FullName != "ИВАНОВ ИВАН" {
		t.Errorf("FullName = %q, want ИВАНОВ ИВАН", id.FullName)
	}
}

func TestExtractIdentityNoIIN(t *testing.T) {
	der := genCert(t, pkix.Name{CommonName: "БЕЗ ИИН", SerialNumber: "X"})
	c, err := pkcs7.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if _, err := extractIdentity(c); !errors.Is(err, ErrNoIIN) {
		t.Fatalf("expected ErrNoIIN, got %v", err)
	}
}

func TestLoadTrustStore(t *testing.T) {
	dir := t.TempDir()
	der := genCert(t, pkix.Name{CommonName: "TEST NUC RK ROOT"})
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, "root.pem"), pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	// второй сертификат в DER
	if err := os.WriteFile(filepath.Join(dir, "root2.cer"), genCert(t, pkix.Name{CommonName: "ROOT2"}), 0o600); err != nil {
		t.Fatal(err)
	}

	ts, err := LoadTrustStore(dir)
	if err != nil {
		t.Fatalf("LoadTrustStore: %v", err)
	}
	if len(ts.CAList) != 2 {
		t.Errorf("CAList = %d, want 2", len(ts.CAList))
	}
	if len(ts.X509Certs) != 2 {
		t.Errorf("X509Certs = %d, want 2", len(ts.X509Certs))
	}
}

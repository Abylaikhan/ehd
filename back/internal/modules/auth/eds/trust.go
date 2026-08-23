package eds

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	pkcs7 "github.com/ddulesov/pkcs7"
)

// TrustStore — доверенные корни/промежуточные сертификаты НУЦ РК.
type TrustStore struct {
	CAList    []*pkcs7.Certificate // для проверки цепочки CMS (RSA + ГОСТ)
	X509Certs []*x509.Certificate  // издатели, пригодные для OCSP (RSA)
}

// LoadTrustStore читает все сертификаты (PEM или DER) из каталога.
func LoadTrustStore(dir string) (*TrustStore, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	ts := &TrustStore{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		for _, der := range decodeCerts(raw) {
			if c, err := pkcs7.ParseCertificate(der); err == nil {
				ts.CAList = append(ts.CAList, c)
			}
			if xc, err := x509.ParseCertificate(der); err == nil {
				ts.X509Certs = append(ts.X509Certs, xc)
			}
		}
	}
	return ts, nil
}

// decodeCerts извлекает DER-блоки: сначала как PEM, иначе трактует файл как DER.
func decodeCerts(raw []byte) [][]byte {
	var out [][]byte
	rest := raw
	for {
		var blk *pem.Block
		blk, rest = pem.Decode(rest)
		if blk == nil {
			break
		}
		if blk.Type == "CERTIFICATE" {
			out = append(out, blk.Bytes)
		}
	}
	if len(out) == 0 && len(raw) > 0 {
		out = append(out, raw)
	}
	return out
}

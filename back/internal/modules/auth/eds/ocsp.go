package eds

import (
	"bytes"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	pkcs7 "github.com/ddulesov/pkcs7"
	"golang.org/x/crypto/ocsp"
)

var ocspClient = &http.Client{Timeout: 5 * time.Second}

// checkRevocation выполняет онлайн OCSP-проверку.
// Требует x509-разбор (RSA); для ГОСТ возвращает ошибку (soft-skip в вызывающем коде).
func (v *NCALayerVerifier) checkRevocation(signer *pkcs7.Certificate) error {
	leaf, err := x509.ParseCertificate(signer.Raw)
	if err != nil {
		return fmt.Errorf("ocsp: сертификат не разбирается как x509 (возможно ГОСТ): %w", err)
	}
	if len(leaf.OCSPServer) == 0 {
		return errors.New("ocsp: в сертификате нет OCSP-респондера (AIA)")
	}
	issuer := v.findIssuer(leaf)
	if issuer == nil {
		return errors.New("ocsp: издатель не найден в trust store")
	}

	reqDER, err := ocsp.CreateRequest(leaf, issuer, nil)
	if err != nil {
		return err
	}
	httpResp, err := ocspClient.Post(leaf.OCSPServer[0], "application/ocsp-request", bytes.NewReader(reqDER))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return err
	}

	resp, err := ocsp.ParseResponseForCert(body, leaf, issuer)
	if err != nil {
		return err
	}
	if resp.Status == ocsp.Revoked {
		return ErrCertificateRevoked
	}
	return nil // Good / Unknown — не отзываем
}

func (v *NCALayerVerifier) findIssuer(leaf *x509.Certificate) *x509.Certificate {
	for _, c := range v.trust.X509Certs {
		if c.Subject.String() == leaf.Issuer.String() {
			return c
		}
	}
	return nil
}

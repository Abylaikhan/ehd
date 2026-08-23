package eds

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	pkcs7 "github.com/ddulesov/pkcs7"
	"go.uber.org/zap"
)

// decodeCMS принимает CMS в PEM-обёртке (как отдаёт NCALayer) или в base64 (std/url) и возвращает DER.
func decodeCMS(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "-----BEGIN") {
		if blk, _ := pem.Decode([]byte(s)); blk != nil {
			return blk.Bytes, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

// NCALayerVerifier — реальная проверка CMS-подписи ЭЦП (NCALayer / НУЦ РК).
// signedData — base64 отделённой (detached) CMS-подписи над challenge.
type NCALayerVerifier struct {
	trust *TrustStore
	ocsp  bool
	log   *zap.Logger
}

func NewNCALayerVerifier(trustDir string, ocsp bool, log *zap.Logger) (*NCALayerVerifier, error) {
	ts, err := LoadTrustStore(trustDir)
	if err != nil {
		return nil, fmt.Errorf("eds: не удалось загрузить trust store из %s: %w", trustDir, err)
	}
	if len(ts.CAList) == 0 {
		return nil, fmt.Errorf("eds: в %s нет доверенных сертификатов НУЦ РК (см. infra/eds-trust/README.md)", trustDir)
	}
	return &NCALayerVerifier{trust: ts, ocsp: ocsp, log: log}, nil
}

func (v *NCALayerVerifier) Verify(challenge, signedData string) (SignatureData, error) {
	sd, err := v.verify(challenge, signedData)
	if err != nil {
		// Причина отказа (без чувствительных данных) — для диагностики.
		v.log.Warn("eds verify failed", zap.Error(err))
	}
	return sd, err
}

func (v *NCALayerVerifier) verify(challenge, signedData string) (SignatureData, error) {
	der, err := decodeCMS(signedData)
	if err != nil {
		v.log.Warn("eds step: decode CMS", zap.Error(err))
		return SignatureData{}, ErrInvalidSignature
	}
	cms, err := pkcs7.ParseCMS(der)
	if err != nil {
		v.log.Warn("eds step: ParseCMS", zap.Error(err), zap.Int("der_len", len(der)))
		return SignatureData{}, ErrInvalidSignature
	}

	now := time.Now()
	// Проверка подписи над challenge (detached) и окна времени подписи.
	if err := cms.Verify([]byte(challenge), now.Add(-time.Hour), now.Add(5*time.Minute)); err != nil {
		v.log.Warn("eds step: cms.Verify(content)", zap.Error(err))
		return SignatureData{}, ErrInvalidSignature
	}
	// Цепочка до доверенного корня НУЦ РК.
	if err := cms.VerifyCertificates(v.trust.CAList); err != nil {
		v.log.Warn("eds step: VerifyCertificates(chain)", zap.Error(err))
		return SignatureData{}, ErrUntrustedCertificate
	}
	if len(cms.Certificates) == 0 {
		return SignatureData{}, ErrInvalidSignature
	}
	signer := cms.Certificates[0]

	// Срок действия сертификата.
	if now.Before(signer.TBSCertificate.Validity.NotBefore) || now.After(signer.TBSCertificate.Validity.NotAfter) {
		return SignatureData{}, ErrCertificateExpired
	}

	id, err := extractIdentity(signer)
	if err != nil {
		return SignatureData{}, err
	}

	// Онлайн-проверка отзыва (OCSP). ГОСТ-ответы stdlib не проверяет — soft-skip с логом.
	if v.ocsp {
		if err := v.checkRevocation(signer); err != nil {
			if errors.Is(err, ErrCertificateRevoked) {
				return SignatureData{}, err
			}
			v.log.Warn("eds ocsp check skipped", zap.String("reason", err.Error()))
		}
	}

	return SignatureData{IIN: id.IIN, BIN: id.BIN, FullName: id.FullName, Valid: true}, nil
}

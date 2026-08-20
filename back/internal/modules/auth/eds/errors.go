package eds

import "errors"

var (
	ErrInvalidSignature     = errors.New("invalid eds signature")
	ErrUntrustedCertificate = errors.New("certificate does not chain to a trusted NUC RK root")
	ErrCertificateExpired   = errors.New("certificate is outside its validity period")
	ErrCertificateRevoked   = errors.New("certificate is revoked")
	ErrNoIIN                = errors.New("iin not found in certificate subject")
)

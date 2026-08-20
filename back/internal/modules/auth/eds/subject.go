package eds

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"regexp"
	"strings"

	pkcs7 "github.com/ddulesov/pkcs7"
)

// Identity — данные субъекта из сертификата НУЦ РК.
type Identity struct {
	IIN      string
	BIN      string
	FullName string
}

var (
	reIIN = regexp.MustCompile(`IIN(\d{12})`)
	reBIN = regexp.MustCompile(`BIN(\d{12})`)
)

// extractIdentity разбирает subject сертификата: ИИН — в SERIALNUMBER (OID 2.5.4.5),
// БИН — у юрлиц (SERIALNUMBER/OU), ФИО — в CN.
func extractIdentity(cert *pkcs7.Certificate) (Identity, error) {
	name, err := subjectName(cert.TBSCertificate.Subject.FullBytes)
	if err != nil {
		return Identity{}, ErrInvalidSignature
	}
	hay := name.SerialNumber + " " + strings.Join(name.OrganizationalUnit, " ")

	id := Identity{FullName: name.CommonName}
	if m := reIIN.FindStringSubmatch(hay); m != nil {
		id.IIN = m[1]
	}
	if m := reBIN.FindStringSubmatch(hay); m != nil {
		id.BIN = m[1]
	}
	// SERIALNUMBER без префикса, но ровно 12 цифр — тоже трактуем как ИИН.
	if id.IIN == "" && len(name.SerialNumber) == 12 && allDigits(name.SerialNumber) {
		id.IIN = name.SerialNumber
	}
	if id.IIN == "" {
		return Identity{}, ErrNoIIN
	}
	return id, nil
}

// subjectName разбирает сырой subject (DER) в pkix.Name.
func subjectName(rawSubject []byte) (pkix.Name, error) {
	var rdn pkix.RDNSequence
	if _, err := asn1.Unmarshal(rawSubject, &rdn); err != nil {
		return pkix.Name{}, err
	}
	var name pkix.Name
	name.FillFromRDNSequence(&rdn)
	return name, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

package eds

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// NCANodeVerifier проверяет CMS через сайдкар NCANode (провайдер Kalkan: RSA + ГОСТ РК).
// signedData — CMS (PEM или base64) как отдаёт NCALayer; challenge — подписанное detached-содержимое.
type NCANodeVerifier struct {
	url    string
	client *http.Client
	log    *zap.Logger
}

func NewNCANodeVerifier(url string, log *zap.Logger) *NCANodeVerifier {
	return &NCANodeVerifier{
		url:    strings.TrimRight(url, "/"),
		client: &http.Client{Timeout: 25 * time.Second},
		log:    log,
	}
}

type ncaVerifyReq struct {
	CMS             string   `json:"cms"`
	Data            string   `json:"data,omitempty"`
	RevocationCheck []string `json:"revocationCheck,omitempty"`
}

type ncaSubject struct {
	CommonName string `json:"commonName"`
	IIN        string `json:"iin"`
	BIN        string `json:"bin"`
}

type ncaCert struct {
	Valid   bool       `json:"valid"`
	Subject ncaSubject `json:"subject"`
}

type ncaSigner struct {
	Certificates []ncaCert `json:"certificates"`
}

type ncaVerifyResp struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Valid   bool        `json:"valid"`
	Signers []ncaSigner `json:"signers"`
}

func (v *NCANodeVerifier) Verify(challenge, signedData string) (SignatureData, error) {
	// NCANode ждёт cms как base64 DER (без PEM-обёртки) — нормализуем то, что отдал NCALayer.
	der, err := decodeCMS(signedData)
	if err != nil {
		return SignatureData{}, ErrInvalidSignature
	}
	cmsB64 := base64.StdEncoding.EncodeToString(der)

	b64 := base64.StdEncoding.EncodeToString([]byte(challenge))
	// NCALayer для detached CMS мог подписать либо сам challenge, либо переданную base64-строку —
	// пробуем обе кодировки data и принимаем ту, что проходит проверку.
	dataCandidates := []string{
		b64, // подписан сырой challenge
		base64.StdEncoding.EncodeToString([]byte(b64)), // подписана base64-строка challenge
	}

	var last *ncaVerifyResp
	for _, data := range dataCandidates {
		r, err := v.callVerify(cmsB64, data)
		if err != nil {
			return SignatureData{}, err
		}
		last = r
		if r.Valid && len(r.Signers) > 0 && len(r.Signers[0].Certificates) > 0 {
			subj := r.Signers[0].Certificates[0].Subject
			if subj.IIN == "" {
				return SignatureData{}, ErrNoIIN
			}
			return SignatureData{IIN: subj.IIN, BIN: subj.BIN, FullName: subj.CommonName, Valid: true}, nil
		}
	}
	msg := ""
	if last != nil {
		msg = last.Message
	}
	v.log.Warn("ncanode verify failed", zap.String("message", msg))
	return SignatureData{}, ErrInvalidSignature
}

func (v *NCANodeVerifier) callVerify(cms, dataB64 string) (*ncaVerifyResp, error) {
	reqBody, err := json.Marshal(ncaVerifyReq{
		CMS:             cms,
		Data:            dataB64,
		RevocationCheck: []string{"OCSP"},
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, v.url+"/cms/verify", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ncanode недоступен: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var r ncaVerifyResp
	if err := json.Unmarshal(body, &r); err != nil {
		v.log.Warn("ncanode: некорректный ответ", zap.ByteString("body", body))
		return &ncaVerifyResp{}, nil
	}
	return &r, nil
}

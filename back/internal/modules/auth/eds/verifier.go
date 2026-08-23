// Package eds — проверка ЭЦП (NCALayer). В sandbox используется StubVerifier;
// реальная проверка сертификата/цепочки/отзыва подключается через тот же интерфейс (Phase 2).
package eds

import "strings"

// SignatureData — извлечённые из подписи данные субъекта.
type SignatureData struct {
	IIN      string
	BIN      string // для юрлица; пусто для физлица
	FullName string
	Valid    bool
}

type Verifier interface {
	// Verify проверяет подпись против выданного challenge и возвращает данные субъекта.
	Verify(challenge, signedData string) (SignatureData, error)
}

// StubVerifier — заглушка sandbox: не выполняет криптопроверку.
// Ожидает signedData вида "IIN|BIN|FullName" (BIN и FullName опциональны).
type StubVerifier struct{}

func (StubVerifier) Verify(_ string, signedData string) (SignatureData, error) {
	parts := strings.SplitN(signedData, "|", 3)
	if len(parts) == 0 || len(parts[0]) != 12 {
		return SignatureData{}, ErrInvalidSignature
	}
	sd := SignatureData{IIN: parts[0], Valid: true}
	if len(parts) >= 2 {
		sd.BIN = parts[1]
	}
	if len(parts) >= 3 {
		sd.FullName = parts[2]
	}
	return sd, nil
}

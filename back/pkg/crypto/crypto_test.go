package crypto

import "testing"

const (
	testKey  = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	testHMAC = "hmac-secret"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	c, err := New(testKey, testHMAC)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	c := newTestCipher(t)
	plain := "990101300123"
	blob, err := c.EncryptString(plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) == plain {
		t.Fatal("ciphertext equals plaintext")
	}
	got, err := c.DecryptString(blob)
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestEncryptNonDeterministic(t *testing.T) {
	c := newTestCipher(t)
	a, _ := c.EncryptString("secret")
	b, _ := c.EncryptString("secret")
	if string(a) == string(b) {
		t.Fatal("expected a random nonce per encryption")
	}
}

func TestEmptyRoundtrip(t *testing.T) {
	c := newTestCipher(t)
	blob, err := c.EncryptString("")
	if err != nil || blob != nil {
		t.Fatalf("empty encrypt: blob=%v err=%v", blob, err)
	}
	s, err := c.DecryptString(nil)
	if err != nil || s != "" {
		t.Fatalf("empty decrypt: s=%q err=%v", s, err)
	}
}

func TestHMACDeterministic(t *testing.T) {
	c := newTestCipher(t)
	if c.HMAC("990101300123") != c.HMAC("990101300123") {
		t.Fatal("HMAC not deterministic")
	}
	if c.HMAC("a") == c.HMAC("b") {
		t.Fatal("HMAC collision on distinct inputs")
	}
}

func TestNewRejectsBadKeys(t *testing.T) {
	if _, err := New("zz", testHMAC); err == nil {
		t.Fatal("expected hex error")
	}
	if _, err := New("00", testHMAC); err == nil {
		t.Fatal("expected 32-byte length error")
	}
	if _, err := New(testKey, ""); err == nil {
		t.Fatal("expected empty hmac error")
	}
}

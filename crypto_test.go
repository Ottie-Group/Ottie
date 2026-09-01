package main

import (
	"testing"
)

func TestCryptoFlow(t *testing.T) {
	password := "correct-horse-battery-staple"
	wrongPassword := "incorrect-password"

	// 1. Generate salt and derive KEK
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	kek, err := DeriveKEK(password, salt)
	if err != nil {
		t.Fatalf("DeriveKEK failed: %v", err)
	}

	// 2. Generate DEK and wrap with KEK
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatalf("GenerateDEK failed: %v", err)
	}

	wrappedDEK, err := WrapDEK(dek, kek)
	if err != nil {
		t.Fatalf("WrapDEK failed: %v", err)
	}

	// 3. Unwrap DEK with correct password
	unwrappedDEK, err := UnwrapDEK(wrappedDEK, kek)
	if err != nil {
		t.Fatalf("UnwrapDEK failed: %v", err)
	}
	if string(unwrappedDEK) != string(dek) {
		t.Fatalf("Unwrapped DEK does not match original DEK")
	}

	// 4. Attempt unwrap with wrong password
	wrongKEK, err := DeriveKEK(wrongPassword, salt)
	if err != nil {
		t.Fatalf("DeriveKEK with wrong password failed: %v", err)
	}
	_, err = UnwrapDEK(wrappedDEK, wrongKEK)
	if err == nil {
		t.Fatalf("Expected UnwrapDEK to fail with wrong KEK, but it succeeded")
	}

	// 5. Encrypt & Decrypt TOTP secret with user DEK
	secret := "JBSWY3DPEHPK3PXP"
	encSecret, err := EncryptSecret(secret, dek)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	decrypted, err := DecryptSecret(encSecret, unwrappedDEK)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("Decrypted secret %q does not match original %q", decrypted, secret)
	}
}

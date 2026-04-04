package server

import "testing"

func TestGenerateToken_Length(t *testing.T) {
	token := generateToken()
	// 16 bytes = 32 hex characters
	if len(token) != 32 {
		t.Errorf("expected token length 32, got %d: %q", len(token), token)
	}
}

func TestGenerateToken_Uniqueness(t *testing.T) {
	a := generateToken()
	b := generateToken()
	if a == b {
		t.Errorf("two tokens should differ, both were %q", a)
	}
}

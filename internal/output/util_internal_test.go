package output

import "testing"

func TestIntToString(t *testing.T) {
	if got, want := intToString(42), "42"; got != want {
		t.Errorf("intToString(42) = %q, want %q", got, want)
	}
}

func TestBoolToString(t *testing.T) {
	if got, want := boolToString(true), "true"; got != want {
		t.Errorf("boolToString(true) = %q, want %q", got, want)
	}
	if got, want := boolToString(false), "false"; got != want {
		t.Errorf("boolToString(false) = %q, want %q", got, want)
	}
}

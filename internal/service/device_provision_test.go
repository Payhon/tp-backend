package service

import "testing"

func TestNormalizeItemUUID(t *testing.T) {
	got := normalizeItemUUID(" 4fAa01bc ")
	if got != "4FAA01BC" {
		t.Fatalf("normalizeItemUUID() = %q, want %q", got, "4FAA01BC")
	}
}

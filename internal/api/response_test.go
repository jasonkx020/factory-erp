package api

import "testing"

func TestFail(t *testing.T) {
	err := Fail("X")
	if err.Error() != "X" {
		t.Fatalf("got %v", err)
	}
}

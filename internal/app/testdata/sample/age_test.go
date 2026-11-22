package sample

import "testing"

func TestIsEligibleOutsideBoundary(t *testing.T) {
	if !IsEligible(20, true) {
		t.Fatal("20-year-old enabled user should be eligible")
	}
}

func TestSame(t *testing.T) {
	if !Same(4, 4) || Same(4, 5) {
		t.Fatal("Same should distinguish equal values")
	}
}

package main

import (
	"strings"
	"testing"
)

func TestDeriveBranchName(t *testing.T) {
	tests := []struct {
		number int
		title  string
		want   string
	}{
		{142, "Fix checkout flow when cart is empty", "142-fix-checkout-flow-when-cart-is-empty"},
		{1, "Short title", "1-short-title"},
		{99, "Title with -- double dashes!!", "99-title-with-double-dashes"},
	}
	for _, tc := range tests {
		got := deriveBranchName(tc.number, tc.title)
		if got != tc.want {
			t.Errorf("deriveBranchName(%d, %q) = %q, want %q", tc.number, tc.title, got, tc.want)
		}
	}
}

func TestDeriveBranchNameMaxLength(t *testing.T) {
	for _, number := range []int{1, 10, 100, 9999} {
		got := deriveBranchName(number, strings.Repeat("a", 100))
		if len(got) > 45 {
			t.Errorf("deriveBranchName(%d, long) = %q (len %d > 45)", number, got, len(got))
		}
	}
}


func TestTruncate(t *testing.T) {
	tests := []struct {
		value string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello w…"},
		{"", 5, ""},
		{"hi", 1, "…"},
		{"hi", 0, ""},
	}
	for _, tc := range tests {
		got := truncate(tc.value, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.value, tc.max, got, tc.want)
		}
	}
}

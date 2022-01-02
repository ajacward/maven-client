package main

import "testing"

func TestDefaultString(t *testing.T) {
	cases := []struct {
		a, b, want string
	}{
		{"text", "default", "text"},
		{"", "default", "default"},
	}
	for _, c := range cases {
		got := defaultString(c.a, c.b)
		if got != c.want {
			t.Errorf("ReverseRunes(%q, %q) == %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

package main

import "testing"

func TestParseGitVersion(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"git version 2.39.1.windows.1", "2.39.1", false},
		{"git version 2.25.1", "2.25.1", false},
		{"git version 2.34.1 (Apple Git-137)", "2.34.1", false},
		{"git version 3.0", "3.0", false},
		{"some unexpected output", "", true},
		{"", "", true},
	}

	for _, c := range cases {
		got, err := ParseGitVersion(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("expected error for input %q, got nil and version %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("ParseGitVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

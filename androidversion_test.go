package main

import "testing"

func TestParseAndroidStudioProductInfo(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{`{"versionName":"2023.1.1"}`, "2023.1.1", false},
		{`{"version":"2023.1.1"}`, "2023.1.1", false},
		{`{"fullVersion":"Android Studio 2023.1.1"}`, "Android Studio 2023.1.1", false},
		{`{"something":"else"}`, "", true},
		{`not-json`, "", true},
	}

	for _, c := range cases {
		got, err := ParseAndroidStudioProductInfo(c.in)
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
			t.Fatalf("ParseAndroidStudioProductInfo(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

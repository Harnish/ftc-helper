package main

import (
	"errors"
	"strings"
	"testing"
)

func TestFriendlyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"dns failure", errors.New("dial tcp: lookup github.com: no such host"), "Couldn't reach the internet. Check your wifi and try again."},
		{"github rate limit", errors.New("unexpected status code: 403"), "GitHub is temporarily limiting requests. Wait a few minutes and try again."},
		{"permission denied", errors.New("open C:\\out.exe: permission denied"), "Windows blocked writing that file. Try running as Administrator, or pick a different folder."},
		{"disk full", errors.New("write out.zip: no space left on device"), "Your disk is full. Free up some space and try again."},
		{"studio not found", errors.New("Could not find Android Studio executable. Set ANDROID_STUDIO_PATH or android_studio_path in config."), "Couldn't find Android Studio. Set ANDROID_STUDIO_PATH or install it first."},
		{"corrupt zip", errors.New("zip: not a valid zip file"), "The download got cut off. Let's try downloading it again."},
		{"unmapped", errors.New("something bizarre happened"), "Something went wrong: something bizarre happened"},
	}

	for _, c := range cases {
		got := friendlyError(c.err)
		if got != c.want {
			t.Errorf("%s: friendlyError(%q) = %q, want %q", c.name, c.err.Error(), got, c.want)
		}
	}
}

func TestFriendlyErrorNil(t *testing.T) {
	if got := friendlyError(nil); got != "" {
		t.Errorf("friendlyError(nil) = %q, want empty string", got)
	}
}

func TestFriendlyErrorNeverEmptyForNonNil(t *testing.T) {
	err := errors.New("x")
	got := friendlyError(err)
	if !strings.Contains(got, "x") {
		t.Errorf("expected fallback to contain original message, got %q", got)
	}
}

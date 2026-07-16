package main

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// stepHeader prints a colored progress header, e.g. "Step 2/5: Installing Android Studio".
func stepHeader(n, total int, label string) {
	color.New(color.FgCyan, color.Bold).Printf("\nStep %d/%d: %s\n", n, total, label)
}

// ok prints a green checkmark line.
func ok(msg string) {
	color.New(color.FgGreen).Println("  ✓ " + msg)
}

// fail prints a red X line.
func fail(msg string) {
	color.New(color.FgRed).Println("  ✗ " + msg)
}

// friendlyError translates common failure modes into plain-English text for
// non-technical users. Unmapped errors fall through to a generic but clearly
// labeled message so nothing fails silently.
func friendlyError(err error) string {
	if err == nil {
		return ""
	}

	lower := strings.ToLower(err.Error())

	switch {
	case strings.Contains(lower, "no such host"), strings.Contains(lower, "dial tcp"), strings.Contains(lower, "timeout"):
		return "Couldn't reach the internet. Check your wifi and try again."
	case strings.Contains(lower, "403"):
		return "GitHub is temporarily limiting requests. Wait a few minutes and try again."
	case strings.Contains(lower, "permission denied"):
		return "Windows blocked writing that file. Try running as Administrator, or pick a different folder."
	case strings.Contains(lower, "no space left"):
		return "Your disk is full. Free up some space and try again."
	case strings.Contains(lower, "could not find android studio executable"):
		return "Couldn't find Android Studio. Set ANDROID_STUDIO_PATH or install it first."
	case strings.Contains(lower, "not a valid zip file"), strings.Contains(lower, "unexpected eof"):
		return "The download got cut off. Let's try downloading it again."
	default:
		return "Something went wrong: " + err.Error()
	}
}

// confirmRetry prompts the user with a yes/no question and returns true for
// an affirmative answer. Any non-"y"/"yes" input (including read failure) is
// treated as "no".
func confirmRetry(scanner *bufio.Scanner, prompt string) bool {
	fmt.Printf("%s (y/n): ", prompt)
	if !scanner.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes"
}

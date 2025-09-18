package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// DetectAndroidStudioVersion tries to locate Android Studio and return its version string.
// Strategy:
// 1. Use findAndroidStudioExe() to locate the studio executable and take its parent directories as install root.
// 2. Look for product-info.json in common locations under the install root and parse it.
// 3. Fallback: look for 'build.txt' or similar files containing version text.
func DetectAndroidStudioVersion() (string, error) {
	exe, err := findAndroidStudioExe()
	if err != nil {
		return "", err
	}

	// Install root is two levels up from bin/<exe>
	// e.g., C:\Program Files\Android\Android Studio\bin\studio64.exe
	installRoot := filepath.Dir(filepath.Dir(exe))

	// Common product-info.json locations
	candidates := []string{
		filepath.Join(installRoot, "product-info.json"),
		filepath.Join(installRoot, "product-info", "product-info.json"),
		filepath.Join(installRoot, "lib", "product-info.json"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			b, err := ioutil.ReadFile(c)
			if err != nil {
				continue
			}
			if v, err := ParseAndroidStudioProductInfo(string(b)); err == nil {
				return v, nil
			}
		}
	}

	// Fallback: look for build.txt under installRoot
	fallback := filepath.Join(installRoot, "build.txt")
	if _, err := os.Stat(fallback); err == nil {
		b, err := ioutil.ReadFile(fallback)
		if err == nil {
			s := strings.TrimSpace(string(b))
			if s != "" {
				return s, nil
			}
		}
	}

	return "", errors.New("could not detect Android Studio version")
}

// ParseAndroidStudioProductInfo parses the JSON content of product-info.json and
// returns either "versionName" or a concatenation of vendor+version.
func ParseAndroidStudioProductInfo(content string) (string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return "", err
	}
	// Typical product-info.json contains "versionName" and/or "version"
	if v, ok := obj["versionName"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}
	if v, ok := obj["version"]; ok {
		switch vv := v.(type) {
		case string:
			if vv != "" {
				return vv, nil
			}
		case float64:
			return fmtFloat(vv), nil
		}
	}
	// Try vendor + full version
	if v, ok := obj["fullVersion"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, nil
		}
	}
	return "", errors.New("no version field found in product-info.json")
}

// fmtFloat formats float without scientific notation and without trailing zeros.
func fmtFloat(f float64) string {
	// Simple formatting
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
	return s
}

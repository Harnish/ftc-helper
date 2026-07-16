package main

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFindAndroidStudioExe_EnvOverride(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "studio-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fakeExe := filepath.Join(tmpDir, "studio.exe")
	if err := os.WriteFile(fakeExe, []byte("dummy"), 0755); err != nil {
		t.Fatalf("failed to write fake exe: %v", err)
	}

	// Set env var and call
	os.Setenv("ANDROID_STUDIO_PATH", fakeExe)
	defer os.Unsetenv("ANDROID_STUDIO_PATH")

	p, err := findAndroidStudioExe()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p != fakeExe {
		t.Fatalf("expected %s, got %s", fakeExe, p)
	}
}

func TestExtractZip(t *testing.T) {
	// create a temp zip file with one file inside
	tmpDir, err := ioutil.TempDir("", "ziptest")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "test.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("hello.txt")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := io.WriteString(w, "hello world"); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	zw.Close()
	f.Close()

	// extract to another temp dir
	dest := filepath.Join(tmpDir, "out")
	if err := extractZip(zipPath, dest); err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	got, err := ioutil.ReadFile(filepath.Join(dest, "hello.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(got) != "hello world" {
		t.Fatalf("extracted content mismatch: %s", string(got))
	}
}

func TestRunInit_SkipsIfProjectAlreadyExists(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "runinit-test")
	if err != nil {
		t.Fatalf("tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origWorkDir := workDir
	workDir = tmpDir
	defer func() { workDir = origWorkDir }()

	teamCodePath := filepath.Join(tmpDir, "myproject", "TeamCode", "src", "main", "java", "org", "firstinspires", "ftc", "teamcode")
	if err := os.MkdirAll(teamCodePath, 0755); err != nil {
		t.Fatalf("mkdir teamCodePath: %v", err)
	}

	// Should return nil immediately without attempting any network download,
	// since the project directory already exists.
	if err := runInit("v11.0", "myproject", ""); err != nil {
		t.Fatalf("expected no error for already-initialized project, got %v", err)
	}
}

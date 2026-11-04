package hostruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverExecutablesPrefersBundledRipgrep(t *testing.T) {
	runtimeName := RipgrepExecutableName()
	bundled := filepath.Join(t.TempDir(), "tools", runtimeName)
	if err := os.MkdirAll(filepath.Dir(bundled), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundled, []byte("bundled ripgrep"), 0o700); err != nil {
		t.Fatal(err)
	}
	pathDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(pathDirectory, runtimeName), []byte("host ripgrep"), 0o700); err != nil {
		t.Fatal(err)
	}

	discovered := DiscoverForExecutableWithEnvironment(
		filepath.Join(filepath.Dir(filepath.Dir(bundled)), "inkdock"),
		[]string{"PATH=" + pathDirectory, "PATHEXT=.EXE"},
		"",
	)
	if discovered.Ripgrep != bundled {
		t.Fatalf("Ripgrep = %q, want bundled %q", discovered.Ripgrep, bundled)
	}
}

func TestDiscoverExecutablesFallsBackToPathRipgrep(t *testing.T) {
	pathDirectory := t.TempDir()
	runtimeName := RipgrepExecutableName()
	hostRipgrep := filepath.Join(pathDirectory, runtimeName)
	if err := os.WriteFile(hostRipgrep, []byte("host ripgrep"), 0o700); err != nil {
		t.Fatal(err)
	}

	discovered := DiscoverForExecutableWithEnvironment(
		filepath.Join(t.TempDir(), "inkdock"),
		[]string{"PATH=" + pathDirectory, "PATHEXT=.EXE"},
		"",
	)
	discoveredInfo, discoveredErr := os.Stat(discovered.Ripgrep)
	wantInfo, wantErr := os.Stat(hostRipgrep)
	if discoveredErr != nil || wantErr != nil || !os.SameFile(discoveredInfo, wantInfo) {
		t.Fatalf("Ripgrep = %q, want host fallback %q", discovered.Ripgrep, hostRipgrep)
	}
}

func TestDiscoverExecutablesReportsMissingRipgrep(t *testing.T) {
	discovered := DiscoverForExecutableWithEnvironment(
		filepath.Join(t.TempDir(), "inkdock"),
		[]string{"PATH=" + t.TempDir(), "PATHEXT=.EXE"},
		"",
	)
	if discovered.Ripgrep != "" {
		t.Fatalf("Ripgrep = %q, want empty when neither bundled nor on PATH", discovered.Ripgrep)
	}
}

func TestDiscoverExecutablesFallsBackFromUnavailableBashOverride(t *testing.T) {
	directory := t.TempDir()
	name := "bash"
	if runtime.GOOS == "windows" {
		name = "bash.exe"
	}
	bashPath := filepath.Join(directory, name)
	if err := os.WriteFile(bashPath, []byte("test executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	environment := []string{"PATH=" + directory, "PATHEXT=.EXE"}
	discovered := DiscoverForExecutableWithEnvironment("", environment, filepath.Join(directory, "missing-bash"))
	discoveredInfo, discoveredErr := os.Stat(discovered.Bash)
	wantInfo, wantErr := os.Stat(bashPath)
	if discoveredErr != nil || wantErr != nil || !os.SameFile(discoveredInfo, wantInfo) {
		t.Fatalf("Bash = %q, want fallback %q", discovered.Bash, bashPath)
	}
}

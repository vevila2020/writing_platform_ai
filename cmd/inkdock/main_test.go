package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/config"
)

func TestFrontendPortProbeRejectsOccupiedPort(t *testing.T) {
	for _, host := range []string{config.LocalHTTPHost, config.LANHTTPHost} {
		t.Run(host, func(t *testing.T) {
			listener, err := listenOnPort(host, "0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			port := listener.Addr().(*net.TCPAddr).Port
			preferred := strconv.Itoa(port)

			t.Run("preferred port", func(t *testing.T) {
				if got := selectFrontendPort(host, preferred); got == preferred {
					t.Fatalf("selected occupied frontend port %s", preferred)
				}
			})
			t.Run("fallback port", func(t *testing.T) {
				if got, err := findAvailablePort(host, strconv.Itoa(port-1), 1); err == nil || got != "" {
					t.Fatalf("fallback frontend port = %s, %v; want no available port", got, err)
				}
			})
		})
	}
}

func TestFrontendPortProbeIgnoresOtherLocalAddresses(t *testing.T) {
	// A listener on another local address must not force a loopback-only
	// frontend to change ports. Probe only the address the frontend will use.
	listener, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("Alternate loopback address is unavailable: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	preferred := strconv.Itoa(port)

	t.Run("preferred port", func(t *testing.T) {
		if got := selectFrontendPort(config.LocalHTTPHost, preferred); got != preferred {
			t.Fatalf("selected frontend port = %s, want %s", got, preferred)
		}
	})
	t.Run("fallback port", func(t *testing.T) {
		got, err := findAvailablePort(config.LocalHTTPHost, strconv.Itoa(port-1), 1)
		if err != nil || got != preferred {
			t.Fatalf("fallback frontend port = %s, %v; want %s", got, err, preferred)
		}
	})
}

func TestReportMissingRipgrepWarnsOnlyInDevelopment(t *testing.T) {
	executablePath := filepath.Join(t.TempDir(), "inkdock")
	environment := []string{"PATH=" + t.TempDir()}

	var releaseOutput bytes.Buffer
	if reportMissingRipgrep(&releaseOutput, false, executablePath, environment) {
		t.Fatal("release runs must not report a missing ripgrep")
	}
	if releaseOutput.Len() != 0 {
		t.Fatalf("release run wrote %q, want no output", releaseOutput.String())
	}

	var developmentOutput bytes.Buffer
	if !reportMissingRipgrep(&developmentOutput, true, executablePath, environment) {
		t.Fatal("development run did not report a missing ripgrep")
	}
	if notice := developmentOutput.String(); !strings.Contains(notice, "ripgrep") {
		t.Fatalf("notice = %q, want it to mention ripgrep", notice)
	}
}

func TestReportMissingRipgrepAcceptsInstalledRipgrep(t *testing.T) {
	directory := t.TempDir()
	name := "rg"
	if runtime.GOOS == "windows" {
		name = "rg.exe"
	}
	if err := os.WriteFile(filepath.Join(directory, name), []byte("ripgrep"), 0o700); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if reportMissingRipgrep(&output, true, filepath.Join(t.TempDir(), "inkdock"), []string{"PATH=" + directory}) {
		t.Fatal("development run reported a missing ripgrep while one is on PATH")
	}
	if output.Len() != 0 {
		t.Fatalf("development run wrote %q, want no output", output.String())
	}
}

func TestSelectFrontendPortSkipsReservedBackendPort(t *testing.T) {
	listener, err := listenOnPort(config.LocalHTTPHost, "0")
	if err != nil {
		t.Fatal(err)
	}
	preferred := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if got := selectFrontendPort(config.LocalHTTPHost, preferred, preferred); got == preferred {
		t.Fatalf("selected reserved backend port %s for the frontend", preferred)
	}
}

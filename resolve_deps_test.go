package resolve_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/git-pkgs/resolve"
	_ "github.com/git-pkgs/resolve/parsers"
)

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func TestResolveDepsUnsupportedManager(t *testing.T) {
	ctx := context.Background()
	_, err := resolve.ResolveDeps(ctx, "nonexistent", nil)
	if !errors.Is(err, resolve.ErrUnsupportedManager) {
		t.Errorf("expected ErrUnsupportedManager, got %v", err)
	}
}

func TestResolveDepsNPM(t *testing.T) {
	if !hasCommand("npm") {
		t.Skip("npm not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := resolve.ResolveDeps(ctx, "npm", []resolve.InputDep{
		{Name: "express", Version: "4.21.2"},
	})
	if err != nil {
		t.Fatalf("ResolveDeps failed: %v", err)
	}

	if result.Manager != "npm" {
		t.Errorf("Manager = %q, want %q", result.Manager, "npm")
	}
	if result.Ecosystem != "npm" {
		t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, "npm")
	}

	express := findDep(result.Direct, "express")
	if express == nil {
		t.Fatal("missing express in results")
	}
	if express.Version != "4.21.2" {
		t.Errorf("express version = %q, want %q", express.Version, "4.21.2")
	}
	// express has 30+ transitive deps
	if len(express.Deps) < 10 {
		t.Errorf("expected express to have 10+ transitive deps, got %d", len(express.Deps))
	}

	// spot check a known transitive dep
	bodyParser := findDep(express.Deps, "body-parser")
	if bodyParser == nil {
		t.Fatal("missing body-parser as transitive dep of express")
	}
}

func TestResolveDepsManagers(t *testing.T) {
	names := resolve.Managers()
	if len(names) == 0 {
		t.Fatal("expected at least one registered manager")
	}
}

func TestResolveDepsEcosystemForManager(t *testing.T) {
	eco, ok := resolve.EcosystemForManager("npm")
	if !ok {
		t.Fatal("expected npm to be registered")
	}
	if eco != "npm" {
		t.Errorf("ecosystem = %q, want %q", eco, "npm")
	}

	eco, ok = resolve.EcosystemForManager("bundler")
	if !ok {
		t.Fatal("expected bundler to be registered")
	}
	if eco != "gem" {
		t.Errorf("ecosystem = %q, want %q", eco, "gem")
	}

	_, ok = resolve.EcosystemForManager("nonexistent")
	if ok {
		t.Error("expected nonexistent manager to not be registered")
	}
}

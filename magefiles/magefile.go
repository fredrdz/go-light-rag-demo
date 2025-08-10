//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// --------------------------------------------------------------------------
// Variables

var (
	goFiles    = "./cmd"
	binaryName = "lrag"
	binDir     = ".builds"
	appData    = "./app_data"
)

// --------------------------------------------------------------------------
// Build functions

// Build namespace
type Build mg.Namespace

// Build for all platforms.
func (Build) All() error {
	mg.Deps(Build.Windows, Build.Linux, Build.Darwin)
	return nil
}

// Build for the current platform.
func (Build) Current() error {
	fmt.Println("Building for the current platform...")
	return build(runtime.GOOS, runtime.GOARCH)
}

// Build for Windows.
func (Build) Windows() error {
	fmt.Println("Building for Windows (amd64)...")
	return build("windows", "amd64")
}

// Build for Linux.
func (Build) Linux() error {
	fmt.Println("Building for Linux (amd64)...")
	return build("linux", "amd64")
}

// Build for macOS.
func (Build) Darwin() error {
	fmt.Println("Building for macOS (amd64)...")
	return build("darwin", "amd64")
}

// Helper function to perform the build.
func build(goos, goarch string) error {
	// create the output directory if it doesn't exist
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// determine the output binary name
	output := filepath.Join(binDir, fmt.Sprintf("%s_%s_%s", binaryName, goos, goarch))
	if goos == "windows" {
		output += ".exe"
	}

	// set environment variables for cross-compilation
	env := map[string]string{
		"GOOS":        goos,
		"GOARCH":      goarch,
		"CGO_ENABLED": "0",
	}

	// execute the build command
	err := sh.RunWith(env, "go", "build", "-ldflags=-s -w", "-o", output, goFiles+"/cli")
	if err != nil {
		return fmt.Errorf("build failed for %s/%s: %w", goos, goarch, err)
	}

	fmt.Printf("Successfully built %s\n", output)
	return nil
}

// --------------------------------------------------------------------------
// Clean functions

// Clean namespace
type Clean mg.Namespace

// Clean binaries and build artifacts.
func (Clean) Binaries() error {
	fmt.Println("Cleaning up binaries...")
	return os.RemoveAll(binDir)
}

// Wipe app data.
func (Clean) WipeData() error {
	fmt.Println("Wiping app data...")
	return os.RemoveAll(appData)
}

// --------------------------------------------------------------------------
// Code functions

// Lint and Test namespace
type Code mg.Namespace

// Run the linter.
func (Code) Lint() error {
	fmt.Println("Running linter...")
	err := sh.RunV("golangci-lint", "run", "-c", "./golangci.yml", "./...")
	if err != nil {
		return err
	}

	fmt.Println("No issues found.")
	return nil
}

// Run ALL tests.
func (Code) Tests() error {
	fmt.Println("Running tests...")
	return sh.RunV("go", "test", "./...", "-v", "--cover")
}

// Run specific test(s) via a string match.
func (Code) Test(match string) error {
	fmt.Printf("Running test(s) with match: %s\n", match)
	return sh.RunV("go", "test", "./...", "-v", "--cover", "-run", match)
}

// Generate test coverage report.
func (Code) TestReport() error {
	fmt.Println("Generating test coverage report...")
	err := sh.RunV("go", "test", "./...", "-v", "--cover", "-coverprofile=coverage.out")
	if err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-html=coverage.out")
}

// --------------------------------------------------------------------------
// Run functions

// Run namespace
type Run mg.Namespace

// Run app via air hot-reload.
func (Run) Air() error {
	fmt.Println("Running application via air...")
	return sh.RunV("air")
}

// Run app as cli with default arguments for testing.
func (Run) Debug() error {
	fmt.Println("Running application as CLI with default arguments...")
	return sh.RunV("go", "run", goFiles+"/cli", "--help")
}

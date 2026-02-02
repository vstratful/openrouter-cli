// Package update provides self-update functionality for the CLI.
package update

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
)

const (
	// Repository owner and name for GitHub releases
	repoOwner = "vstratful"
	repoName  = "openrouter-cli"
)

// ErrDevVersion is returned when trying to update a development build.
var ErrDevVersion = errors.New("cannot update development builds")

// Release contains information about an available update.
type Release struct {
	Version     string
	ReleaseURL  string
	ReleaseDate string
	Description string
	AssetURL    string
	AssetName   string

	// internal reference to the actual release object
	release *selfupdate.Release
}

// newUpdater creates a configured selfupdate.Updater with GitHub source and checksum validation.
func newUpdater() (*selfupdate.Updater, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	return updater, nil
}

// CheckForUpdate checks if a newer version is available on GitHub Releases.
// Returns nil if already on the latest version or if currentVersion is "dev".
func CheckForUpdate(ctx context.Context, currentVersion string) (*Release, error) {
	if currentVersion == "dev" {
		return nil, ErrDevVersion
	}

	updater, err := newUpdater()
	if err != nil {
		return nil, err
	}

	release, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		return nil, fmt.Errorf("failed to detect latest release: %w", err)
	}

	if !found {
		return nil, nil
	}

	// Compare versions - the library handles this
	if !release.GreaterThan(currentVersion) {
		return nil, nil
	}

	releaseDate := ""
	if !release.PublishedAt.IsZero() {
		releaseDate = release.PublishedAt.Format("2006-01-02")
	}

	return &Release{
		Version:     release.Version(),
		ReleaseURL:  release.URL,
		ReleaseDate: releaseDate,
		Description: release.ReleaseNotes,
		AssetURL:    release.AssetURL,
		AssetName:   release.AssetName,
		release:     release,
	}, nil
}

// ApplyUpdate downloads and applies the update, replacing the current binary.
func ApplyUpdate(ctx context.Context, rel *Release) error {
	if rel == nil || rel.release == nil {
		return errors.New("no release to apply")
	}

	updater, err := newUpdater()
	if err != nil {
		return err
	}

	// Get the path to the current executable
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, rel.release, exe); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	return nil
}

// GetPlatformInfo returns the current OS and architecture.
func GetPlatformInfo() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}

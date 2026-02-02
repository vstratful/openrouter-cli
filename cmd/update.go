package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vstratful/openrouter-cli/internal/update"
)

var (
	checkOnly     bool
	forceUpdate   bool
	updateTimeout time.Duration
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the CLI to the latest version",
	Long: `Check for and install updates from GitHub Releases.

Examples:
  openrouter update              # Check and install update interactively
  openrouter update --check      # Only check for updates
  openrouter update --force      # Update without confirmation
  openrouter update --timeout 60s # Set network timeout`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Update without confirmation")
	updateCmd.Flags().DurationVar(&updateTimeout, "timeout", 30*time.Second, "Timeout for network operations")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()

	currentVersion := version
	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n", currentVersion)

	release, err := update.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		if errors.Is(err, update.ErrDevVersion) {
			fmt.Println("\nYou are running a development build.")
			fmt.Println("Auto-update is only available for released versions.")
			fmt.Println("Install a release from: https://github.com/vstratful/openrouter-cli/releases")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release == nil {
		fmt.Println("\nYou are running the latest version.")
		return nil
	}

	fmt.Printf("Latest version:  %s\n", release.Version)

	if release.Description != "" {
		fmt.Printf("\nRelease notes:\n")
		// Indent release notes for readability
		lines := strings.Split(release.Description, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	}

	if checkOnly {
		fmt.Printf("\nRun 'openrouter update' to install the update.\n")
		return nil
	}

	if !forceUpdate {
		fmt.Printf("\nDo you want to update? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	fmt.Printf("\nDownloading %s...\n", release.AssetName)

	// Cancel the check context before creating a new one for download
	cancel()
	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), updateTimeout*2)
	defer downloadCancel()

	if err := update.ApplyUpdate(downloadCtx, release); err != nil {
		// Note: go-selfupdate doesn't export typed errors for permission/checksum failures,
		// so we fall back to string matching. This is fragile but necessary.
		errMsg := err.Error()
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access is denied") {
			fmt.Println("\nPermission denied. Try running with elevated privileges:")
			osName, _ := update.GetPlatformInfo()
			if osName == "windows" {
				fmt.Println("  Run as Administrator")
			} else {
				fmt.Println("  sudo openrouter update")
			}
			return err
		}
		if strings.Contains(errMsg, "checksum") {
			fmt.Println("\nSecurity warning: Checksum verification failed!")
			fmt.Println("The downloaded file may be corrupted or tampered with.")
			fmt.Println("Please download manually from: https://github.com/vstratful/openrouter-cli/releases")
			return err
		}
		return err
	}

	fmt.Printf("\nSuccessfully updated to v%s!\n", release.Version)
	return nil
}

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/registry"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Automatically add 'tm' to Windows User PATH environment variable",
	Long: `Copies the current tm.exe executable to '%LOCALAPPDATA%\\Programs\\tm\\tm.exe' 
and appends this directory to the User PATH environment variable in the Windows registry.
This allows running 'tm' directly from any newly opened CMD or PowerShell window.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}

		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}

		targetDir := filepath.Join(localAppData, "Programs", "tm")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		targetExe := filepath.Join(targetDir, "tm.exe")

		// Copy executable
		srcFile, err := os.Open(exePath)
		if err != nil {
			return fmt.Errorf("failed to open source executable: %w", err)
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetExe)
		if err != nil {
			return fmt.Errorf("failed to create target executable: %w", err)
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy executable: %w", err)
		}

		// Register to User Environment PATH
		key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.ALL_ACCESS)
		if err != nil {
			return fmt.Errorf("failed to open Windows registry: %w", err)
		}
		defer key.Close()

		currentPath, _, err := key.GetStringValue("Path")
		if err != nil {
			currentPath = ""
		}

		paths := filepath.SplitList(currentPath)
		alreadyInPath := false
		for _, p := range paths {
			if strings.EqualFold(filepath.Clean(p), filepath.Clean(targetDir)) {
				alreadyInPath = true
				break
			}
		}

		if !alreadyInPath {
			var newPath string
			if currentPath == "" {
				newPath = targetDir
			} else {
				newPath = currentPath + string(os.PathListSeparator) + targetDir
			}

			if err := key.SetStringValue("Path", newPath); err != nil {
				return fmt.Errorf("failed to update PATH: %w", err)
			}
			fmt.Println("✓ PATH environment variable updated successfully.")
		} else {
			fmt.Println("ℹ This directory is already in your PATH.")
		}

		fmt.Println("---------------------------------------------------------------")
		fmt.Printf("✓ Installation complete! Binary location: %s\n", targetExe)
		fmt.Println("★ You can now open any new CMD or PowerShell window and type:")
		fmt.Println("  'tm' to launch TaskMaster!")
		fmt.Println("---------------------------------------------------------------")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

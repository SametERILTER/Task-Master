package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"tm/internal/metrics"
)

var killCmd = &cobra.Command{
	Use:   "kill [PID]",
	Short: "Directly terminate a process by PID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PID: %s", args[0])
		}

		err = metrics.KillProcess(int32(pid))
		if err != nil {
			return err
		}

		fmt.Printf("✓ PID %d terminated successfully.\n", pid)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}

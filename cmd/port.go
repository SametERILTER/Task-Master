package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"tm/internal/metrics"
)

var portCmd = &cobra.Command{
	Use:   "port [number]",
	Short: "Show open listening ports and their owning processes",
	Long: `Lists active listening (LISTEN) network ports and associated processes.
If a port number is provided, shows only the process using that specific port.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPort uint32
		if len(args) > 0 {
			p, err := strconv.Atoi(args[0])
			if err != nil || p <= 0 || p > 65535 {
				return fmt.Errorf("invalid port number: %s (must be between 1 and 65535)", args[0])
			}
			targetPort = uint32(p)
		}

		return printPortList(targetPort)
	},
}

func printPortList(targetPort uint32) error {
	ports, err := metrics.GetOpenPorts(targetPort)
	if err != nil {
		return err
	}

	if len(ports) == 0 {
		if targetPort > 0 {
			fmt.Printf("ℹ No active process found listening on port %d.\n", targetPort)
		} else {
			fmt.Println("ℹ No active listening ports found.")
		}
		return nil
	}

	fmt.Printf("%-7s %-8s %-18s %-8s %s\n", "PORT", "PROTO", "ADDRESS", "PID", "PROCESS NAME")
	fmt.Println("----------------------------------------------------------------------")
	for _, p := range ports {
		fmt.Printf("%-7d %-8s %-18s %-8d %s\n", p.Port, p.Protocol, p.LocalAddr, p.PID, p.ProcessName)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(portCmd)
}

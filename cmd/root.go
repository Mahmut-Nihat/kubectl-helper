package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// 🟣 RootCmd dışa açık olmalı ve plugin olduğumuz için Hidden: true
var RootCmd = &cobra.Command{
	Use:    "helper", // plugin adın
	Hidden: true,     // böylece kubectl normalde listemez, sadece plugin çağırır
	Short:  "Helper commands for kubectl",
	Long:   `Helper commands for kubectl operations.`,
}

func Execute() {
	// ip komutunu ekliyoruz
	RootCmd.AddCommand(ipCmd)

	// root bir iş yapmasın sadece alt komutları çalıştırsın
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

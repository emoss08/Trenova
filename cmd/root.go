package cmd

import (
	"fmt"
	"os"

	"github.com/emoss08/trenova/config"
	"github.com/spf13/cobra"
)

var configEnv string

var rootCmd = &cobra.Command{
	Use:   "trenova",
	Short: "Trenova is a transportation management system",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.SetConfigEnv(configEnv)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configEnv, "env", "", "configuration environment (e.g., dev, prod)")
}

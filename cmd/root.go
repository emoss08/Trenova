// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
		env, _ := cmd.Flags().GetString("env")
		config.SetConfigEnv(env)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("env", "", "configuration environment (default is dev)")
}

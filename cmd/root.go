// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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

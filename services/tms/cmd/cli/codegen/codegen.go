package codegen

import (
	"github.com/spf13/cobra"
)

var CodegenCmd = &cobra.Command{
	Use:   "codegen",
	Short: "Code generation commands",
	Long: `Generate code from permission registry definitions.
This includes generating TypeScript types, Go enums, and other artifacts.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate all code artifacts",
	Long:  `Generate all code artifacts (Go enums, TypeScript types, etc.)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")

		if err := GenerateResourceEnum(output); err != nil {
			return err
		}

		if err := GenerateTypeScriptTypes(output); err != nil {
			return err
		}

		return nil
	},
}

var generateEnumCmd = &cobra.Command{
	Use:   "enum",
	Short: "Generate Go Resource enum from registry",
	Long:  `Generate the Resource enum in Go based on registered permission entities`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		return GenerateResourceEnum(output)
	},
}

var generateTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "Generate TypeScript types from registry",
	Long:  `Generate TypeScript types and metadata from field definitions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		return GenerateTypeScriptTypes(output)
	},
}

func init() {
	generateCmd.Flags().StringP("output", "o", ".", "Output directory for generated files")
	generateEnumCmd.Flags().StringP("output", "o", ".", "Output directory for generated files")
	generateTypesCmd.Flags().StringP("output", "o", ".", "Output directory for generated files")

	CodegenCmd.AddCommand(generateCmd)
	CodegenCmd.AddCommand(generateEnumCmd)
	CodegenCmd.AddCommand(generateTypesCmd)
}
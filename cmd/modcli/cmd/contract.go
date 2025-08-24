package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CrisisTextLine/modular/cmd/modcli/internal/contract"
	"github.com/spf13/cobra"
)

// Define static errors
var (
	ErrUnsupportedFormat = errors.New("unsupported output format")
)

// NewContractCommand creates the contract command
func NewContractCommand() *cobra.Command {
	// Local flag variables to avoid global state issues in tests
	var (
		outputFile      string
		includePrivate  bool
		includeTests    bool
		includeInternal bool
		outputFormat    string
		ignorePositions bool
		ignoreComments  bool
		verbose         bool
	)

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "API contract management for Go packages",
		Long: `The contract command provides functionality to extract, compare, and manage
API contracts for Go packages. This helps detect breaking changes and track
API evolution over time.

Available subcommands:
  extract  - Extract API contract from a Go package
  compare  - Compare two API contracts and show differences
  diff     - Alias for compare command`,
	}

	// Create extract command with local flag variables
	extractCmd := &cobra.Command{
		Use:   "extract [package]",
		Short: "Extract API contract from a Go package",
		Long: `Extract the public API contract from a Go package or directory.
The contract includes exported interfaces, types, functions, variables, and constants.

Examples:
  modcli contract extract .                     # Current directory
  modcli contract extract ./modules/auth       # Specific directory
  modcli contract extract github.com/user/pkg  # Remote package
  modcli contract extract -o contract.json .   # Save to file`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtractContractWithFlags(cmd, args, outputFile, includePrivate, includeTests, includeInternal, verbose)
		},
	}

	// Create compare command with local flag variables
	compareCmd := &cobra.Command{
		Use:   "compare <old-contract> <new-contract>",
		Short: "Compare two API contracts",
		Long: `Compare two API contract files and show the differences.
This command identifies breaking changes, additions, and modifications.

Examples:
  modcli contract compare old.json new.json
  modcli contract compare old.json new.json -o diff.json
  modcli contract compare old.json new.json --format=markdown`,
		Args:    cobra.ExactArgs(2),
		Aliases: []string{"diff"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompareContractWithFlags(cmd, args, outputFile, outputFormat, ignorePositions, ignoreComments, verbose)
		},
	}

	// Setup extract command flags
	extractCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	extractCmd.Flags().BoolVar(&includePrivate, "include-private", false, "Include unexported items")
	extractCmd.Flags().BoolVar(&includeTests, "include-tests", false, "Include test files")
	extractCmd.Flags().BoolVar(&includeInternal, "include-internal", false, "Include internal packages")
	extractCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Setup compare command flags
	compareCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	compareCmd.Flags().StringVar(&outputFormat, "format", "json", "Output format: json, markdown, text")
	compareCmd.Flags().BoolVar(&ignorePositions, "ignore-positions", true, "Ignore source position changes")
	compareCmd.Flags().BoolVar(&ignoreComments, "ignore-comments", false, "Ignore documentation comment changes")
	compareCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	contractCmd.AddCommand(extractCmd)
	contractCmd.AddCommand(compareCmd)
	return contractCmd
}

func runExtractContractWithFlags(cmd *cobra.Command, args []string, outputFile string, includePrivate bool, includeTests bool, includeInternal bool, verbose bool) error {
	packagePath := args[0]

	if verbose {
		fmt.Fprintf(os.Stderr, "Extracting API contract from: %s\n", packagePath)
	}

	extractor := contract.NewExtractor()
	extractor.IncludePrivate = includePrivate
	extractor.IncludeTests = includeTests
	extractor.IncludeInternal = includeInternal

	var apiContract *contract.Contract
	var err error

	// Check if it's a local directory
	if strings.HasPrefix(packagePath, ".") || strings.HasPrefix(packagePath, "/") {
		// Resolve relative paths
		if absPath, err := filepath.Abs(packagePath); err == nil {
			packagePath = absPath
		}
		apiContract, err = extractor.ExtractFromDirectory(packagePath)
	} else {
		// Treat as a package path
		apiContract, err = extractor.ExtractFromPackage(packagePath)
	}

	if err != nil {
		return fmt.Errorf("failed to extract contract: %w", err)
	}

	// Output the contract
	if outputFile != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Saving contract to: %s\n", outputFile)
		}

		if err := apiContract.SaveToFile(outputFile); err != nil {
			return fmt.Errorf("failed to save contract: %w", err)
		}

		fmt.Printf("API contract extracted and saved to %s\n", outputFile)
	} else {
		// Output to stdout as pretty JSON
		data, err := json.MarshalIndent(apiContract, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal contract: %w", err)
		}
		fmt.Println(string(data))
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Contract extracted successfully:\n")
		fmt.Fprintf(os.Stderr, "  - Package: %s\n", apiContract.PackageName)
		fmt.Fprintf(os.Stderr, "  - Interfaces: %d\n", len(apiContract.Interfaces))
		fmt.Fprintf(os.Stderr, "  - Types: %d\n", len(apiContract.Types))
		fmt.Fprintf(os.Stderr, "  - Functions: %d\n", len(apiContract.Functions))
		fmt.Fprintf(os.Stderr, "  - Variables: %d\n", len(apiContract.Variables))
		fmt.Fprintf(os.Stderr, "  - Constants: %d\n", len(apiContract.Constants))
	}

	return nil
}

func runCompareContractWithFlags(cmd *cobra.Command, args []string, outputFile string, outputFormat string, ignorePositions bool, ignoreComments bool, verbose bool) error {
	oldFile := args[0]
	newFile := args[1]

	if verbose {
		fmt.Fprintf(os.Stderr, "Comparing contracts: %s -> %s\n", oldFile, newFile)
	}

	// Load contracts
	oldContract, err := contract.LoadFromFile(oldFile)
	if err != nil {
		return fmt.Errorf("failed to load old contract: %w", err)
	}

	newContract, err := contract.LoadFromFile(newFile)
	if err != nil {
		return fmt.Errorf("failed to load new contract: %w", err)
	}

	// Compare contracts
	differ := contract.NewDiffer()
	differ.IgnorePositions = ignorePositions
	differ.IgnoreComments = ignoreComments

	diff, err := differ.Compare(oldContract, newContract)
	if err != nil {
		return fmt.Errorf("failed to compare contracts: %w", err)
	}

	// Format and output the diff
	var output string
	switch strings.ToLower(outputFormat) {
	case "json":
		output, err = formatDiffAsJSON(diff)
	case "markdown", "md":
		output, err = formatDiffAsMarkdown(diff)
	case "text", "txt":
		output, err = formatDiffAsText(diff)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, outputFormat)
	}

	if err != nil {
		return fmt.Errorf("failed to format diff: %w", err)
	}

	// Output the diff
	if outputFile != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Saving diff to: %s\n", outputFile)
		}

		if err := os.WriteFile(outputFile, []byte(output), 0600); err != nil {
			return fmt.Errorf("failed to save diff: %w", err)
		}

		fmt.Printf("Contract diff saved to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Comparison completed:\n")
		fmt.Fprintf(os.Stderr, "  - Breaking changes: %d\n", diff.Summary.TotalBreakingChanges)
		fmt.Fprintf(os.Stderr, "  - Additions: %d\n", diff.Summary.TotalAdditions)
		fmt.Fprintf(os.Stderr, "  - Modifications: %d\n", diff.Summary.TotalModifications)
	}

	// Exit with error code if there are breaking changes
	if diff.Summary.HasBreakingChanges {
		fmt.Fprintf(os.Stderr, "WARNING: Breaking changes detected!\n")
		os.Exit(1)
	}

	return nil
}

func formatDiffAsJSON(diff *contract.ContractDiff) (string, error) {
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal diff as JSON: %w", err)
	}
	return string(data), nil
}

func formatDiffAsMarkdown(diff *contract.ContractDiff) (string, error) {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# API Contract Diff: %s\n\n", diff.PackageName))

	if diff.OldVersion != "" || diff.NewVersion != "" {
		md.WriteString("## Version Information\n")
		if diff.OldVersion != "" {
			md.WriteString(fmt.Sprintf("- **Old Version**: %s\n", diff.OldVersion))
		}
		if diff.NewVersion != "" {
			md.WriteString(fmt.Sprintf("- **New Version**: %s\n", diff.NewVersion))
		}
		md.WriteString("\n")
	}

	// Summary
	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Breaking Changes**: %d\n", diff.Summary.TotalBreakingChanges))
	md.WriteString(fmt.Sprintf("- **Additions**: %d\n", diff.Summary.TotalAdditions))
	md.WriteString(fmt.Sprintf("- **Modifications**: %d\n", diff.Summary.TotalModifications))

	if diff.Summary.HasBreakingChanges {
		md.WriteString("\n⚠️  **Warning: This update contains breaking changes!**\n")
	}
	md.WriteString("\n")

	// Breaking changes
	if len(diff.BreakingChanges) > 0 {
		md.WriteString("## 🚨 Breaking Changes\n\n")
		for _, change := range diff.BreakingChanges {
			md.WriteString(fmt.Sprintf("### %s: %s\n", change.Type, change.Item))
			md.WriteString(fmt.Sprintf("%s\n\n", change.Description))
			if change.OldValue != "" {
				md.WriteString("**Old:**\n```go\n")
				md.WriteString(change.OldValue)
				md.WriteString("\n```\n\n")
			}
			if change.NewValue != "" {
				md.WriteString("**New:**\n```go\n")
				md.WriteString(change.NewValue)
				md.WriteString("\n```\n\n")
			}
		}
	}

	// Additions
	if len(diff.AddedItems) > 0 {
		md.WriteString("## ➕ Additions\n\n")
		for _, item := range diff.AddedItems {
			md.WriteString(fmt.Sprintf("- **%s**: %s - %s\n", item.Type, item.Item, item.Description))
		}
		md.WriteString("\n")
	}

	// Modifications
	if len(diff.ModifiedItems) > 0 {
		md.WriteString("## 📝 Modifications\n\n")
		for _, item := range diff.ModifiedItems {
			md.WriteString(fmt.Sprintf("- **%s**: %s - %s\n", item.Type, item.Item, item.Description))
		}
		md.WriteString("\n")
	}

	return md.String(), nil
}

func formatDiffAsText(diff *contract.ContractDiff) (string, error) {
	var txt strings.Builder

	txt.WriteString(fmt.Sprintf("API Contract Diff: %s\n", diff.PackageName))
	txt.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Summary
	txt.WriteString("Summary:\n")
	txt.WriteString(fmt.Sprintf("  Breaking Changes: %d\n", diff.Summary.TotalBreakingChanges))
	txt.WriteString(fmt.Sprintf("  Additions: %d\n", diff.Summary.TotalAdditions))
	txt.WriteString(fmt.Sprintf("  Modifications: %d\n", diff.Summary.TotalModifications))

	if diff.Summary.HasBreakingChanges {
		txt.WriteString("\n*** WARNING: Breaking changes detected! ***\n")
	}
	txt.WriteString("\n")

	// Breaking changes
	if len(diff.BreakingChanges) > 0 {
		txt.WriteString("BREAKING CHANGES:\n")
		txt.WriteString(strings.Repeat("-", 20) + "\n")
		for _, change := range diff.BreakingChanges {
			txt.WriteString(fmt.Sprintf("- %s: %s\n", change.Type, change.Item))
			txt.WriteString(fmt.Sprintf("  %s\n", change.Description))
			if change.OldValue != "" && change.NewValue != "" {
				txt.WriteString(fmt.Sprintf("  Old: %s\n", change.OldValue))
				txt.WriteString(fmt.Sprintf("  New: %s\n", change.NewValue))
			}
			txt.WriteString("\n")
		}
	}

	// Additions
	if len(diff.AddedItems) > 0 {
		txt.WriteString("ADDITIONS:\n")
		txt.WriteString(strings.Repeat("-", 20) + "\n")
		for _, item := range diff.AddedItems {
			txt.WriteString(fmt.Sprintf("+ %s: %s\n", item.Type, item.Item))
		}
		txt.WriteString("\n")
	}

	// Modifications
	if len(diff.ModifiedItems) > 0 {
		txt.WriteString("MODIFICATIONS:\n")
		txt.WriteString(strings.Repeat("-", 20) + "\n")
		for _, item := range diff.ModifiedItems {
			txt.WriteString(fmt.Sprintf("~ %s: %s\n", item.Type, item.Item))
		}
		txt.WriteString("\n")
	}

	return txt.String(), nil
}

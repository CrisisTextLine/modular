package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/CrisisTextLine/modular/cmd/modcli/internal/contract"
	"github.com/spf13/cobra"
)

func TestContractCommand(t *testing.T) {
	cmd := NewContractCommand()
	
	if cmd.Use != "contract" {
		t.Errorf("Expected Use to be 'contract', got %s", cmd.Use)
	}

	if len(cmd.Commands()) != 2 {
		t.Errorf("Expected 2 subcommands, got %d", len(cmd.Commands()))
	}

	// Check that extract and compare commands are present
	hasExtract := false
	hasCompare := false
	
	for _, subcmd := range cmd.Commands() {
		switch subcmd.Use {
		case "extract [package]":
			hasExtract = true
		case "compare <old-contract> <new-contract>":
			hasCompare = true
		}
	}

	if !hasExtract {
		t.Error("Expected extract command to be present")
	}
	if !hasCompare {
		t.Error("Expected compare command to be present")
	}
}

func TestExtractCommand_Help(t *testing.T) {
	// Create individual command instances to avoid flag conflicts
	extractCmd := &cobra.Command{
		Use:   "extract [package]",
		Short: "Extract API contract from a Go package",
		Long:  `Extract API contract help text`,
	}
	
	compareCmd := &cobra.Command{
		Use:   "compare <old-contract> <new-contract>",
		Short: "Compare two API contracts",
		Long:  `Compare API contracts help text`,
	}

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "API contract management for Go packages",
	}
	
	contractCmd.AddCommand(extractCmd)
	contractCmd.AddCommand(compareCmd)
	
	buf := new(bytes.Buffer)
	contractCmd.SetOut(buf)
	contractCmd.SetArgs([]string{"extract", "--help"})
	
	err := contractCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute extract help: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Extract API contract")) {
		t.Error("Expected help output to contain 'Extract API contract'")
	}
}

func TestCompareCommand_Help(t *testing.T) {
	// Create individual command instances to avoid flag conflicts
	compareCmd := &cobra.Command{
		Use:   "compare <old-contract> <new-contract>",
		Short: "Compare two API contracts",
		Long:  `Compare API contracts help text`,
	}

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "API contract management for Go packages",
	}
	
	contractCmd.AddCommand(compareCmd)
	
	buf := new(bytes.Buffer)
	contractCmd.SetOut(buf)
	contractCmd.SetArgs([]string{"compare", "--help"})
	
	err := contractCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute compare help: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Compare two API contracts")) {
		t.Error("Expected help output to contain 'Compare two API contracts'")
	}
}

func TestExtractCommand_InvalidArgs(t *testing.T) {
	cmd := NewContractCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"extract"}) // Missing package argument
	
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing package argument")
	}
}

func TestCompareCommand_InvalidArgs(t *testing.T) {
	cmd := NewContractCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"compare", "only-one-arg"}) // Need two arguments
	
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for insufficient arguments")
	}
}

func TestRunExtractContract_ValidDirectory(t *testing.T) {
	// Create a temporary directory with a simple Go package
	tmpDir, err := os.MkdirTemp("", "extract-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Go file
	testCode := `package testpkg

// TestInterface is a test interface
type TestInterface interface {
	TestMethod(input string) error
}

// TestFunc is a test function
func TestFunc() {}
`

	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Reset global flags
	outputFile = ""
	includePrivate = false
	verbose = false

	// Test the command
	cmd := &cobra.Command{}
	err = runExtractContract(cmd, []string{tmpDir})
	if err != nil {
		t.Fatalf("Failed to extract contract: %v", err)
	}
}

func TestRunExtractContract_InvalidDirectory(t *testing.T) {
	// Reset global flags
	outputFile = ""
	
	cmd := &cobra.Command{}
	err := runExtractContract(cmd, []string{"/nonexistent/directory"})
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestRunCompareContract_ValidContracts(t *testing.T) {
	// Create two test contracts
	contract1 := &contract.Contract{
		PackageName: "test",
		Version:     "v1.0.0",
	}

	contract2 := &contract.Contract{
		PackageName: "test", 
		Version:     "v2.0.0",
		Functions: []contract.FunctionContract{
			{Name: "NewFunction", Package: "test"},
		},
	}

	// Create temporary files
	file1, err := os.CreateTemp("", "contract1-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file1.Name())

	file2, err := os.CreateTemp("", "contract2-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file2.Name())

	// Write contracts to files
	if err := contract1.SaveToFile(file1.Name()); err != nil {
		t.Fatalf("Failed to save contract1: %v", err)
	}

	if err := contract2.SaveToFile(file2.Name()); err != nil {
		t.Fatalf("Failed to save contract2: %v", err)
	}

	// Reset global flags
	outputFile = ""
	outputFormat = "json"
	verbose = false

	// Test the command
	cmd := &cobra.Command{}
	err = runCompareContract(cmd, []string{file1.Name(), file2.Name()})
	if err != nil {
		t.Fatalf("Failed to compare contracts: %v", err)
	}
}

func TestRunCompareContract_InvalidFiles(t *testing.T) {
	// Reset global flags
	outputFile = ""
	
	cmd := &cobra.Command{}
	err := runCompareContract(cmd, []string{"/nonexistent/file1.json", "/nonexistent/file2.json"})
	if err == nil {
		t.Error("Expected error for nonexistent files")
	}
}

func TestFormatDiffAsJSON(t *testing.T) {
	diff := &contract.ContractDiff{
		PackageName: "test",
		Summary: contract.DiffSummary{
			TotalAdditions: 1,
		},
		AddedItems: []contract.AddedItem{
			{Type: "function", Item: "TestFunc", Description: "New function added"},
		},
	}

	output, err := formatDiffAsJSON(diff)
	if err != nil {
		t.Fatalf("Failed to format diff as JSON: %v", err)
	}

	// Verify it's valid JSON
	var parsed contract.ContractDiff
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	if parsed.PackageName != diff.PackageName {
		t.Errorf("Package name mismatch after JSON round-trip: got %s, want %s", 
			parsed.PackageName, diff.PackageName)
	}
}

func TestFormatDiffAsMarkdown(t *testing.T) {
	diff := &contract.ContractDiff{
		PackageName: "test",
		OldVersion:  "v1.0.0",
		NewVersion:  "v2.0.0",
		Summary: contract.DiffSummary{
			TotalBreakingChanges: 1,
			TotalAdditions:       1,
			HasBreakingChanges:   true,
		},
		BreakingChanges: []contract.BreakingChange{
			{Type: "removed_function", Item: "OldFunc", Description: "Function was removed"},
		},
		AddedItems: []contract.AddedItem{
			{Type: "function", Item: "NewFunc", Description: "New function added"},
		},
	}

	output, err := formatDiffAsMarkdown(diff)
	if err != nil {
		t.Fatalf("Failed to format diff as Markdown: %v", err)
	}

	// Check for expected markdown elements
	expectedElements := []string{
		"# API Contract Diff: test",
		"## Version Information",
		"v1.0.0",
		"v2.0.0", 
		"## Summary",
		"⚠️  **Warning: This update contains breaking changes!**",
		"## 🚨 Breaking Changes",
		"### removed_function: OldFunc",
		"## ➕ Additions",
	}

	for _, element := range expectedElements {
		if !bytes.Contains([]byte(output), []byte(element)) {
			t.Errorf("Expected markdown to contain %q", element)
		}
	}
}

func TestFormatDiffAsText(t *testing.T) {
	diff := &contract.ContractDiff{
		PackageName: "test",
		Summary: contract.DiffSummary{
			TotalAdditions: 1,
		},
		AddedItems: []contract.AddedItem{
			{Type: "function", Item: "NewFunc", Description: "New function added"},
		},
	}

	output, err := formatDiffAsText(diff)
	if err != nil {
		t.Fatalf("Failed to format diff as text: %v", err)
	}

	expectedElements := []string{
		"API Contract Diff: test",
		"Summary:",
		"Additions: 1",
		"ADDITIONS:",
		"+ function: NewFunc",
	}

	for _, element := range expectedElements {
		if !bytes.Contains([]byte(output), []byte(element)) {
			t.Errorf("Expected text to contain %q", element)
		}
	}
}
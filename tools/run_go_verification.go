package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/shell"
)

// RunGoVerificationTool returns the MCP tool definition for run_go_verification.
func RunGoVerificationTool() mcp.Tool {
	return mcp.NewTool("run_go_verification",
		mcp.WithDescription("Run Go quality checks: go fmt, go vet, and go test ./... sequentially. Returns detailed logs so the LLM can identify and auto-fix issues. Stops on first failure."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("work_dir",
			mcp.Description("Working directory of the Go project (default: current directory)"),
		),
		mcp.WithString("test_args",
			mcp.Description("Additional arguments for go test (e.g. -v -run TestFoo)"),
		),
	)
}

// HandleRunGoVerification returns a handler that runs Go quality checks.
func HandleRunGoVerification() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workDir := request.GetString("work_dir", ".")
		testArgs := request.GetString("test_args", "")

		// Build the test command with optional extra args.
		testCmd := []string{"go", "test", "./..."}
		if testArgs != "" {
			testCmd = append(testCmd, strings.Fields(testArgs)...)
		}

		commands := [][]string{
			{"go", "fmt", "./..."},
			{"go", "vet", "./..."},
			testCmd,
		}

		results, err := shell.RunAll(ctx, workDir, commands)
		if err != nil {
			return mcp.NewToolResultError("Failed to execute verification: " + err.Error()), nil
		}

		var report strings.Builder
		report.WriteString("## Go Verification Report\n\n")
		report.WriteString(shell.FormatResults(results))

		allPassed := true
		for _, r := range results {
			if !r.IsSuccess() {
				allPassed = false
				break
			}
		}

		if allPassed {
			report.WriteString(fmt.Sprintf("\n**All %d checks passed.**", len(results)))
		} else {
			passed := 0
			for _, r := range results {
				if r.IsSuccess() {
					passed++
				}
			}
			report.WriteString(fmt.Sprintf("\n**%d/%d checks passed.** Fix the failing step and re-run.", passed, len(results)))
		}

		return mcp.NewToolResultText(report.String()), nil
	}
}

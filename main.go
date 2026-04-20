package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mekari/easysdlc/bitbucket"
	"github.com/mekari/easysdlc/instructions"
	"github.com/mekari/easysdlc/resources"
	"github.com/mekari/easysdlc/tools"
)

func main() {
	bbToken := os.Getenv("BITBUCKET_TOKEN")
	if bbToken == "" {
		fmt.Fprintln(os.Stderr, "Error: BITBUCKET_TOKEN environment variable is required.")
		os.Exit(1)
	}
	bbClient := bitbucket.NewClient(bbToken)

	s := server.NewMCPServer(
		"sdlc-bridge",
		"2.0.0",
		server.WithToolCapabilities(false),
	)

	// --- Tools: Bitbucket PR review ---
	s.AddTool(tools.GetRecentPRsTool(), tools.HandleGetRecentPRs(bbClient))
	s.AddTool(tools.ReadPRContentTool(), tools.HandleReadPRContent(bbClient))
	s.AddTool(tools.SubmitPRReviewTool(), tools.HandleSubmitPRReview(bbClient))
	s.AddTool(tools.ReviewOpenPRsTool(), tools.HandleReviewOpenPRs(bbClient))

	// --- Tools: SDLC workflow ---
	s.AddTool(tools.SetupBitbucketBranchTool(), tools.HandleSetupBitbucketBranch(bbClient))
	s.AddTool(tools.RunGoVerificationTool(), tools.HandleRunGoVerification())
	s.AddTool(tools.SubmitBitbucketPRTool(), tools.HandleSubmitBitbucketPR(bbClient))

	// --- Resources ---
	s.AddResourceTemplate(resources.PRListResource(), resources.HandlePRListResource(bbClient))
	s.AddResourceTemplate(resources.PRDetailResource(), resources.HandlePRDetailResource(bbClient))

	// --- Prompts ---
	s.AddPrompt(instructions.ReviewPRPrompt(), instructions.HandleReviewPRPrompt())
	s.AddPrompt(instructions.SummarizeRecentPRsPrompt(), instructions.HandleSummarizeRecentPRsPrompt())
	s.AddPrompt(instructions.BatchCodeReviewPrompt(), instructions.HandleBatchCodeReviewPrompt())
	s.AddPrompt(instructions.SDLCWorkflowPrompt(), instructions.HandleSDLCWorkflowPrompt())
	s.AddPrompt(instructions.FullCopilotPrompt(), instructions.HandleFullCopilotPrompt())

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

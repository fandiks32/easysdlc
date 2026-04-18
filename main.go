package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mekari/easysdlc/bitbucket"
	"github.com/mekari/easysdlc/confluence"
	"github.com/mekari/easysdlc/instructions"
	"github.com/mekari/easysdlc/resources"
	"github.com/mekari/easysdlc/tools"
)

func main() {
	// --- Required: Bitbucket ---
	bbToken := os.Getenv("BITBUCKET_TOKEN")
	if bbToken == "" {
		fmt.Fprintln(os.Stderr, "Error: BITBUCKET_TOKEN environment variable is required.")
		os.Exit(1)
	}
	bbClient := bitbucket.NewClient(bbToken)

	// --- Optional: Confluence ---
	var cfClient *confluence.Client
	cfBaseURL := os.Getenv("CONFLUENCE_BASE_URL")
	cfEmail := os.Getenv("CONFLUENCE_EMAIL")
	cfToken := os.Getenv("CONFLUENCE_TOKEN")
	if cfBaseURL != "" && cfEmail != "" && cfToken != "" {
		cfClient = confluence.NewClient(cfBaseURL, cfEmail, cfToken)
		fmt.Fprintln(os.Stderr, "Confluence integration enabled.")
	} else {
		fmt.Fprintln(os.Stderr, "Confluence integration disabled (set CONFLUENCE_BASE_URL, CONFLUENCE_EMAIL, CONFLUENCE_TOKEN to enable).")
	}

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

	// --- Tools: SDLC workflow (new) ---
	s.AddTool(tools.SetupBitbucketBranchTool(), tools.HandleSetupBitbucketBranch(bbClient))
	s.AddTool(tools.RunGoVerificationTool(), tools.HandleRunGoVerification())
	s.AddTool(tools.SubmitBitbucketPRTool(), tools.HandleSubmitBitbucketPR(bbClient))

	if cfClient != nil {
		s.AddTool(tools.FetchConfluenceRFCTool(), tools.HandleFetchConfluenceRFC(cfClient))
	}

	// --- Resources ---
	s.AddResourceTemplate(resources.PRListResource(), resources.HandlePRListResource(bbClient))
	s.AddResourceTemplate(resources.PRDetailResource(), resources.HandlePRDetailResource(bbClient))

	if cfClient != nil {
		s.AddResourceTemplate(resources.ConfluenceRFCResource(), resources.HandleConfluenceRFCResource(cfClient))
	}

	// --- Prompts ---
	s.AddPrompt(instructions.ReviewPRPrompt(), instructions.HandleReviewPRPrompt())
	s.AddPrompt(instructions.SummarizeRecentPRsPrompt(), instructions.HandleSummarizeRecentPRsPrompt())
	s.AddPrompt(instructions.BatchCodeReviewPrompt(), instructions.HandleBatchCodeReviewPrompt())
	s.AddPrompt(instructions.SDLCWorkflowPrompt(), instructions.HandleSDLCWorkflowPrompt())

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

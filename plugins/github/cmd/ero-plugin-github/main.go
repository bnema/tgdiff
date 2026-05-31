package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	gh "github.com/cli/go-gh/v2"

	"ero/pkg/plugin"
)

const providerID = "github"

type githubProvider struct {
	getenv func(string) string
	execGH func(context.Context, ...string) (string, string, error)
}

type ghPR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type ghReviewResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
}

func main() {
	provider := githubProvider{getenv: os.Getenv}
	if err := plugin.ServeReviewProvider(context.Background(), provider, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (p githubProvider) Initialize(_ context.Context, req plugin.InitializeRequest) (plugin.InitializeResult, error) {
	if req.Protocol != plugin.ProtocolVersion {
		return plugin.InitializeResult{}, plugin.NewErrorf(plugin.ErrorInvalidRequest, "unsupported protocol %q", req.Protocol)
	}
	if req.ContributionID != "" && req.ContributionID != providerID {
		return plugin.InitializeResult{}, plugin.NewErrorf(plugin.ErrorInvalidRequest, "unsupported contribution %q", req.ContributionID)
	}
	return plugin.InitializeResult{
		Protocol: plugin.ProtocolVersion,
		Provider: plugin.ReviewProviderInfo{
			ID:    providerID,
			Label: "GitHub",
			Name:  "ero-plugin-github",
			Capabilities: plugin.ReviewProviderCapabilities{
				LoadRemoteComments: false,
				PublishReview:      true,
				Decisions: []plugin.ReviewDecision{
					plugin.ReviewDecisionComment,
					plugin.ReviewDecisionRequestChanges,
					plugin.ReviewDecisionApprove,
				},
				IdempotentPublish: true,
			},
		},
	}, nil
}

func (p githubProvider) DetectContext(_ context.Context, req plugin.DetectContextRequest) (plugin.DetectContextResult, error) {
	for _, remote := range req.Context.Repository.Remotes {
		if isGitHubRemote(remote.URL) {
			return plugin.DetectContextResult{Result: plugin.DetectionResult{Applicable: true, Reason: "GitHub remote detected"}}, nil
		}
	}
	return plugin.DetectContextResult{Result: plugin.DetectionResult{Applicable: false, Reason: "no GitHub remote detected"}}, nil
}

func (p githubProvider) LoadRemoteThreads(_ context.Context, _ plugin.LoadRemoteThreadsRequest) (plugin.LoadRemoteThreadsResult, error) {
	return plugin.LoadRemoteThreadsResult{}, plugin.NewError(plugin.ErrorUnsupportedCapability, "GitHub remote comment loading is not implemented yet")
}

func (p githubProvider) PublishReview(ctx context.Context, req plugin.PublishReviewParams) (plugin.PublishReviewResultData, error) {
	if p.execGH == nil {
		p.execGH = execGH
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pr, err := p.currentPullRequest(ctx)
	if err != nil {
		return plugin.PublishReviewResultData{}, err
	}
	args, err := buildReviewArgs(pr.Number, req.Payload)
	if err != nil {
		return plugin.PublishReviewResultData{}, err
	}
	stdout, stderr, err := p.execGH(ctx, args...)
	if err != nil {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = err.Error()
		}
		return plugin.PublishReviewResultData{}, plugin.NewErrorf(plugin.ErrorNetwork, "publish GitHub review: %s", message)
	}
	var response ghReviewResponse
	_ = json.Unmarshal([]byte(stdout), &response)
	return plugin.PublishReviewResultData{Result: plugin.ReviewPublishResult{
		ProviderID:       providerID,
		ExternalReviewID: githubReviewID(response, pr),
		ExternalURL:      firstNonEmpty(response.HTMLURL, pr.URL),
		PublishedRefs:    githubPublishedRefs(req.Payload.Draft.Comments),
	}}, nil
}

func (p githubProvider) currentPullRequest(ctx context.Context) (ghPR, error) {
	if p.execGH == nil {
		p.execGH = execGH
	}
	stdout, stderr, err := p.execGH(ctx, "pr", "view", "--json", "number,url")
	if err != nil {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = "no pull request associated with the current branch"
		}
		return ghPR{}, plugin.NewErrorf(plugin.ErrorNotApplicable, "no pull request found for current branch: %s", message)
	}
	var pr ghPR
	if err := json.Unmarshal([]byte(stdout), &pr); err != nil || pr.Number == 0 {
		return ghPR{}, plugin.NewError(plugin.ErrorNotApplicable, "no pull request found for current branch")
	}
	return pr, nil
}

func buildReviewArgs(prNumber int, payload plugin.ReviewPublishPayload) ([]string, error) {
	args := []string{"api", "-X", "POST", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews", prNumber)}
	if payload.Context.Repository.HeadSHA != "" {
		args = append(args, "-f", "commit_id="+payload.Context.Repository.HeadSHA)
	}
	event := githubReviewEvent(payload.Draft.Decision)
	args = append(args, "-f", "event="+event)
	body := strings.TrimSpace(payload.Draft.Summary)
	if body == "" && len(payload.Draft.Comments) == 0 {
		body = "Ero review published."
	}
	if body != "" {
		args = append(args, "-f", "body="+body)
	}
	for _, comment := range payload.Draft.Comments {
		commentArgs, err := githubCommentArgs(comment)
		if err != nil {
			return nil, err
		}
		args = append(args, commentArgs...)
	}
	return args, nil
}

func githubCommentArgs(comment plugin.ReviewComment) ([]string, error) {
	body := strings.TrimSpace(comment.Body)
	if body == "" {
		return nil, plugin.NewError(plugin.ErrorInvalidRequest, "GitHub review comment body is empty")
	}
	line, side, ok := githubLineAndSide(comment.Range.End)
	if !ok {
		return nil, plugin.NewErrorf(plugin.ErrorInvalidRequest, "GitHub review comment %s has no mappable end line", comment.ID)
	}
	args := []string{
		"-f", "comments[][path]=" + comment.FilePath,
		"-f", "comments[][body]=" + body,
		"-F", "comments[][line]=" + strconv.Itoa(line),
		"-f", "comments[][side]=" + side,
	}
	if startLine, startSide, ok := githubLineAndSide(comment.Range.Start); ok && startLine != line {
		args = append(args, "-F", "comments[][start_line]="+strconv.Itoa(startLine), "-f", "comments[][start_side]="+startSide)
	}
	return args, nil
}

func githubLineAndSide(ref plugin.ReviewLineRef) (int, string, bool) {
	if ref.NewLineNumber > 0 {
		return ref.NewLineNumber, "RIGHT", true
	}
	if ref.OldLineNumber > 0 {
		return ref.OldLineNumber, "LEFT", true
	}
	return 0, "", false
}

func githubReviewEvent(decision plugin.ReviewDecision) string {
	switch decision {
	case plugin.ReviewDecisionApprove:
		return "APPROVE"
	case plugin.ReviewDecisionRequestChanges:
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

func githubPublishedRefs(comments []plugin.ReviewComment) []plugin.PublishedReviewCommentRef {
	refs := make([]plugin.PublishedReviewCommentRef, len(comments))
	for i, comment := range comments {
		refs[i] = plugin.PublishedReviewCommentRef{LocalCommentID: comment.ID}
	}
	return refs
}

func githubReviewID(response ghReviewResponse, pr ghPR) string {
	if response.ID > 0 {
		return strconv.FormatInt(response.ID, 10)
	}
	return fmt.Sprintf("pr-%d", pr.Number)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func execGH(ctx context.Context, args ...string) (string, string, error) {
	stdout, stderr, err := gh.ExecContext(ctx, args...)
	return stdout.String(), stderr.String(), err
}

func isGitHubRemote(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, "github.com:") || strings.Contains(url, "github.com/")
}

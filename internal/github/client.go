package github

import (
	"context"
	"fmt"

	"github-file-sync/internal/log"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
}

// NewClient for github with authentication configured.
func NewClient(ctx context.Context, ghToken string) (Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	c := github.NewClient(tc)
	return Client{c}, nil
}

// GetAuthenticatedUsername return the username of the current authenticated user.
func (c Client) GetAuthenticatedUsername(ctx context.Context) (string, error) {
	user, resp, err := c.Client.Users.Get(ctx, "") // empty string makes the library returning the authenticated user
	if err != nil {
		return "", fmt.Errorf("getting user: %v", err)
	}
	defer resp.Body.Close()
	if user == nil {
		return "", fmt.Errorf("retrieved a nil user")
	}
	if user.Login == nil {
		return "", fmt.Errorf("retrieved an empty login")
	}
	return *user.Login, nil
}

// GetHeadBranchNameByPRNumbers for a given repository as a map. Consider only opened PRs.
func (c Client) GetHeadBranchNameByPRNumbers(ctx context.Context, owner, repoName string) (map[int]string, error) {
	// max page size is 100: https://docs.github.com/en/rest/pulls/pulls?apiVersion=2022-11-28#list-pull-requests
	opt := &github.PullRequestListOptions{State: "all", ListOptions: github.ListOptions{PerPage: 99}}

	prs, resp, err := c.Client.PullRequests.List(ctx, owner, repoName, opt)
	if err != nil {
		return nil, fmt.Errorf("listing prs: %v", err)
	}

	defer resp.Body.Close()

	// log a warning if the number of PR retrieved is the maximum page size
	if len(prs) == 99 { //nolint:gomnd
		log.Warnf("99 opened PRs on this repository, this may make the synchronization to fail")
	}

	headBranchNameByPRNumbers := make(map[int]string, len(prs))
	for _, pr := range prs {
		if pr.Head.Ref != nil && pr.Number != nil {
			headBranchNameByPRNumbers[*pr.Number] = *pr.Head.Ref
		}
	}
	return headBranchNameByPRNumbers, nil
}

// CreateOrUpdatePR according to the existingPRNumber parameter.
// On update, the desc is added to the Pull Request as a comment.
func (c Client) CreateOrUpdatePR(
	ctx context.Context, existingPRNumber *int,
	owner, repoName,
	baseBranch, headBranch,
	title, desc string,
) error {
	if existingPRNumber == nil { // create mode
		canBeModified := true
		pr := &github.NewPullRequest{
			Title:               &title,
			Base:                &baseBranch,
			Head:                &headBranch,
			Body:                &desc,
			MaintainerCanModify: &canBeModified,
		}
		createdPR, resp, err := c.Client.PullRequests.Create(ctx, owner, repoName, pr)
		if err != nil {
			return fmt.Errorf("creating PR: %s", err)
		}
		defer resp.Body.Close()
		log.Infof("PR created: %s", *createdPR.HTMLURL)
	} else { // update mode = create a comment with the given desc
		desc = fmt.Sprintf("PR updated with additional changes: %s", desc)
		prComment, resp, err := c.Client.Issues.CreateComment(ctx, owner, repoName, *existingPRNumber, &github.IssueComment{
			Body: &desc,
		})
		if err != nil {
			return fmt.Errorf("creating comment on PR: %v", err)
		}
		defer resp.Body.Close()
		log.Infof("PR updated: %s", *prComment.HTMLURL)
	}
	return nil
}

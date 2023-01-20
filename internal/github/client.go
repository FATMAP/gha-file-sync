package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
}

func NewClient(ctx context.Context, ghToken string) (Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	c := github.NewClient(tc)
	return Client{c}, nil
}

func (c Client) GetBranchNamesFromPRs(ctx context.Context, owner, repoName string) ([]string, error) {
	opt := &github.PullRequestListOptions{State: "open"}

	// TODO: check if pagination is mandatory to implement day 1
	prs, _, err := c.Client.PullRequests.List(ctx, owner, repoName, opt)
	if err != nil {
		return nil, fmt.Errorf("listing prs: %v", err)
	}

	branchNames := []string{}
	for _, pr := range prs {
		if pr.Head.Ref != nil {
			branchNames = append(branchNames, *pr.Head.Ref)
		}
	}
	return branchNames, nil
}

func (c Client) CreatePR(ctx context.Context, owner, repoName string) error {
	// pr := &github.NewPullRequest{
	// 	Title               *string `json:"title,omitempty"`
	// 	Head                *string `json:"head,omitempty"`
	// 	Base                *string `json:"base,omitempty"`
	// 	Body                *string `json:"body,omitempty"`
	// 	Issue               *int    `json:"issue,omitempty"`
	// 	MaintainerCanModify *bool   `json:"maintainer_can_modify,omitempty"
	// }
	// _, _, err := c.PullRequests.Create(ctx, owner, repoName, pr)
	// return err
	return nil
}

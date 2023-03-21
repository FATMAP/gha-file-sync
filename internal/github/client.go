package github

import (
	"context"
	"fmt"
	"strings"

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

func (c Client) CreateOrUpdatePR(ctx context.Context, repoName, baseBranch, prTitle string, createPRMode bool) error {
	desc := "this is the desc"
	if createPRMode {
		canBeModified := true
		pr := &github.NewPullRequest{
			Title:               &prTitle,
			Base:                &baseBranch,
			Body:                &desc,
			MaintainerCanModify: &canBeModified,
		}
		repoSplit := strings.Split(repoName, "/")
		_, _, err := c.PullRequests.Create(ctx, repoSplit[0], repoSplit[1], pr)
		return err
	} else {
		fmt.Printf("Update MODE: WHAT SHOULD I DO BUDDY?")
	}
	return nil
}

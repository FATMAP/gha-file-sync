package github

import (
	"context"
	"fmt"
	"git-file-sync/internal/log"

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

func (c Client) GetBranchNameByPRNumbers(ctx context.Context, owner, repoName string) (map[int]string, error) {
	opt := &github.PullRequestListOptions{State: "open"}

	// TODO: check if pagination is mandatory to implement day 1
	prs, _, err := c.Client.PullRequests.List(ctx, owner, repoName, opt)
	if err != nil {
		return nil, fmt.Errorf("listing prs: %v", err)
	}

	branchNameByPRNumbers := make(map[int]string, len(prs))
	for _, pr := range prs {
		if pr.Head.Ref != nil && pr.Number != nil {
			branchNameByPRNumbers[*pr.Number] = *pr.Head.Ref
		}
	}
	return branchNameByPRNumbers, nil
}

func (c Client) CreateOrUpdatePR(
	ctx context.Context, existingPRNumber *int,
	owner, repoName,
	baseBranch, syncBranch,
	title, desc string,
) error {
	prURL := "unexpected-unset-pr-url"
	if existingPRNumber == nil { // create mode
		canBeModified := true
		pr := &github.NewPullRequest{
			Title:               &title,
			Base:                &baseBranch,
			Head:                &syncBranch,
			Body:                &desc,
			MaintainerCanModify: &canBeModified,
		}
		createdPR, _, err := c.Client.PullRequests.Create(ctx, owner, repoName, pr)
		if err != nil {
			return fmt.Errorf("creating PR: %v", err)
		}
		prURL = *createdPR.HTMLURL
	} else { // update mode = create a comment with the given desc
		fmt.Println("Issue number (pr): ", *existingPRNumber)
		// prComment, _, err := c.Client.Issues.CreateComment(ctx, owner, repoName, *existingPRNumber, &github.IssueComment{
		// 	Body: &desc,
		// })
		// if err != nil {
		// 	return fmt.Errorf("create comment on PR: %v", err)
		// }
		// prURL = *prComment.HTMLURL
	}
	log.Infof("changed push on PR %s", prURL)
	return nil
}

package main

import (
	"context"
	"fmt"
	"log"

	"git-file-sync/github"
)

func main() {
	ctx := context.Background()

	c, err := initConfig()
	if err != nil {
		log.Fatalf("initing config: %v", err)
	}
	fmt.Printf("Configuration:\n %+v", c)

	for _, repoName := range c.RepositoryNames {
		if err := syncRepository(ctx, c, repoName); err != nil {
			log.Printf("syncing %s: %v", repoName, err)
		}
	}
}

func syncRepository(ctx context.Context, c Config, repoName string) error {
	log.Printf("Syncing %s...", repoName)

	rm := github.NewRepoManager(repoName, c.Workspace, c.GithubURL, c.GithubToken, c.FileSyncBranchRegexp)

	// ensure we clean data at the end of the sync
	defer func() {
		err := rm.CleanAll()
		if err != nil {
			log.Printf("ERROR: cleaning %s: %v", rm.RepoName, err)
		}
	}()

	// clone the repo to local filesystem
	err := rm.Clone(ctx)
	if err != nil {
		return fmt.Errorf("cloning: %v", err)
	}

	// set the final branch to compare with - could be an existing file-sync pull request or the main branch
	err = rm.PickBranchToCompare()
	if err != nil {
		return fmt.Errorf("picking branch to compare: %v", err)
	}

	// show the diff
	hasDiffered, err := rm.HasDiffered()
	if err != nil {
		return fmt.Errorf("creating diff: %v", err)
	}

	if hasDiffered && !c.IsDryRun {
		// TODO: create or update the pull request if something has change
		if err := rm.CreateOrUpdateFileSyncPR(); err != nil {
			return fmt.Errorf("creating or updating file sync pr: %v", err)
			// no continue - we try to clean anyway: next step
		}
	}
	return nil
}

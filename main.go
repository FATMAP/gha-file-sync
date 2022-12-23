package main

import (
	"context"
	"fmt"
	"log"

	"git-file-sync/internal/cfg"
	"git-file-sync/internal/github"
)

func main() {
	ctx := context.Background()

	log.Println("~")
	log.Println("Configuring...")
	c, err := cfg.InitConfig()
	if err != nil {
		log.Fatalf("initing config: %v", err)
	}
	c.Print()

	log.Println("~")
	log.Println("Let's sync!")
	for _, repoName := range c.RepositoryNames {
		if err := syncRepository(ctx, c, repoName); err != nil {
			log.Printf("syncing %s: %v", repoName, err)
		}
	}
}

func syncRepository(ctx context.Context, c cfg.Config, repoName string) error {
	log.Printf("> syncing %s...", repoName)

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
	diff, err := rm.GetDiff()
	if err != nil {
		return fmt.Errorf("creating diff: %v", err)
	}

	if diff == "" {
		log.Println("diff detected!")
		log.Println(diff)
		if c.IsDryRun {
			log.Println("-> dry run: nothing pushed for real.")
		} else {
			if err := rm.CreateOrUpdateFileSyncPR(); err != nil {
				return fmt.Errorf("creating or updating file sync pr: %v", err)
				// no continue - we try to clean anyway: next step
			}
		}
	} else {
		log.Println("-> nothing has changed.")
	}
	return nil
}

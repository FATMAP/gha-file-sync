package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	// init github client
	ghClient, err := github.NewClient(ctx, c.GithubToken)
	if err != nil {
		log.Fatalf("initing github client: %v", err)
	}

	for _, repoName := range c.RepositoryNames {
		if err := syncRepository(ctx, c, ghClient, repoName); err != nil {
			log.Printf("syncing %s: %v", repoName, err)
		}
	}
}

func syncRepository(ctx context.Context, c cfg.Config, ghClient github.Client, repoFullname string) error {
	log.Printf("> syncing %s...", repoFullname)

	repoFullnameSplit := strings.Split(repoFullname, "/")
	owner := repoFullnameSplit[0]
	repoName := repoFullnameSplit[1]

	rm := github.NewRepoManager(
		repoName, owner, c.Workspace, c.GithubURL, c.GithubToken, c.FileSyncBranchRegexp,
		ghClient,
		c.FilesBindings,
	)

	// ensure we clean data at the end of the sync
	defer func() {
		err := rm.CleanAll(ctx)
		if err != nil {
			log.Printf("ERROR: cleaning %s: %s", repoFullname, err)
		}
	}()

	// clone the repo to local filesystem
	err := rm.Clone(ctx)
	if err != nil {
		return fmt.Errorf("cloning: %v", err)
	}

	// set the head branch to compare - to see if something has changed
	err = rm.SetHeadBranch(ctx)
	if err != nil {
		return fmt.Errorf("picking branch to compare: %v", err)
	}

	// check if status reports changes
	hasChanged, err := rm.HasChangedAfterCopy(ctx)
	if err != nil {
		return fmt.Errorf("has changed: %v", err)
	}

	if hasChanged {
		log.Println("-> it has changed!")
		log.Println(hasChanged)
		if c.IsDryRun {
			log.Println("-> dry run: nothing pushed for real.")
		} else {
			if err := rm.CreateOrUpdateFileSyncPR(ctx); err != nil {
				return fmt.Errorf("creating or updating file sync pr: %v", err)
				// no continue - we try to clean anyway: next step
			}
		}
	} else {
		log.Println("-> nothing has changed.")
	}
	return nil
}

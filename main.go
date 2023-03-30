package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"git-file-sync/internal/cfg"
	"git-file-sync/internal/github"
	"git-file-sync/internal/log"
	"git-file-sync/internal/sync"
)

func main() {
	ctx := context.Background()

	// init of logger
	log.Init()

	// init of config
	log.Infof("Configuration...")
	c, err := cfg.InitConfig()
	if err != nil {
		log.Errorf("initing config: %v", err)
		os.Exit(1)
	}
	c.Print()

	// init of github client
	ghClient, err := github.NewClient(ctx, c.GithubToken)
	if err != nil {
		log.Errorf("initing github client: %v", err)
		os.Exit(1)
	}

	// start
	log.Infof("Let's sync")
	for _, repoName := range c.RepositoryNames {
		// TODO: make it async?
		if err := syncRepository(ctx, c, ghClient, repoName); err != nil {
			log.Errorf("syncing %s: %v", repoName, err)
		}
	}
	log.Infof("Sync finished.")
}

func syncRepository(ctx context.Context, c cfg.Config, ghClient github.Client, repoFullname string) error {
	log.Infof("Syncing %s...", repoFullname)

	repoFullnameSplit := strings.Split(repoFullname, "/")
	owner := repoFullnameSplit[0]
	repoName := repoFullnameSplit[1]

	t, err := sync.NewTask(
		ctx,
		owner, repoName,
		c.Workspace,
		c.GithubURL, c.GithubToken, ghClient,
		c.FileSyncBranchRegexp,
		c.FilesBindings,
	)
	if err != nil {
		return fmt.Errorf("create repo manager: %v", err)
	}

	// ensure we clean data at the end of the sync
	defer func() {
		err := t.CleanAll(ctx)
		if err != nil {
			log.Errorf("cleaning %s: %v", repoFullname, err)
		}
	}()

	// compute the sync branch to contribute on
	err = t.PickSyncBranch(ctx)
	if err != nil {
		return fmt.Errorf("picking base branch to compare: %v", err)
	}

	// check if anything has changed
	hasChanged, err := t.HasChangedAfterCopy(ctx)
	if err != nil {
		return fmt.Errorf("has changed: %v", err)
	}

	if hasChanged {
		log.Infof("-> it has changed!")
		if c.IsDryRun {
			log.Infof("-> dry run: no concrete write action.")
		} else {
			if err := t.UpdateRemote(ctx, c.CommitMessage, c.PRTitle); err != nil {
				return fmt.Errorf("update remote repo: %v", err)
			}
		}
	} else {
		log.Infof("-> nothing has changed.")
	}
	return nil
}

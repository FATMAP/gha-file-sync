package sync

import (
	"context"
	"fmt"
	"strings"

	"github-file-sync/internal/cfg"
	"github-file-sync/internal/github"
	"github-file-sync/internal/log"
)

// Do synchronize one repository
func Do(ctx context.Context, repoFullname string, c cfg.Config, ghClient github.Client) error {
	log.Infof("Syncing %s...", repoFullname)

	repoFullnameSplit := strings.Split(repoFullname, "/")
	owner := repoFullnameSplit[0]
	repoName := repoFullnameSplit[1]

	t, err := NewTask(
		ctx,
		owner, repoName,
		c.FileSourcePath, c.Workspace,
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
	// could be a new or existing one
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

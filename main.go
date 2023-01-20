package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"git-file-sync/internal/cfg"
	"git-file-sync/internal/github"
)

func main() {
	ctx := context.Background()

	// init of logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// init of config
	log.Print("~")
	log.Print("Configuring...")
	c, err := cfg.InitConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("initing config")
	}
	c.Print()

	// init of github client
	ghClient, err := github.NewClient(ctx, c.GithubToken)
	if err != nil {
		log.Fatal().Err(err).Msg("initing github client")
	}

	// start
	log.Print("~")
	log.Print("Let's sync!")
	for _, repoName := range c.RepositoryNames {
		// TODO: make it async?
		if err := syncRepository(ctx, c, ghClient, repoName); err != nil {
			log.Error().Err(err).Msgf("syncing %s", repoName)
		}
	}
	log.Print("Sync finished.")
}

func syncRepository(ctx context.Context, c cfg.Config, ghClient github.Client, repoFullname string) error {
	log.Info().Msgf("> syncing %s...", repoFullname)

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
			log.Error().Err(err).Msgf("cleaning %s", repoFullname)
		}
	}()

	// clone the repo to local filesystem
	err := rm.Clone(ctx)
	if err != nil {
		return fmt.Errorf("cloning: %v", err)
	}

	// set the base branch to compare with to see if something has changed
	err = rm.PickBaseBranch(ctx)
	if err != nil {
		return fmt.Errorf("picking base branch to compare: %v", err)
	}

	// check if status reports changes
	hasChanged, err := rm.HasChangedAfterCopy(ctx)
	if err != nil {
		return fmt.Errorf("has changed: %v", err)
	}

	if hasChanged {
		log.Info().Msg("-> it has changed!")
		log.Info().Bool("hasChanged", hasChanged)
		// if c.IsDryRun {
		// log.Info().Msg("-> dry run: nothing pushed for real.")
		// } else {
		if err := rm.UpdateRemote(ctx, c.CommitMessage); err != nil {
			return fmt.Errorf("creating or updating file sync pr: %v", err)
		}
		// }
	} else {
		log.Info().Msg("-> nothing has changed.")
	}
	return nil
}

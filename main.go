package main

import (
	"context"
	"os"

	"gha-file-sync/internal/cfg"
	"gha-file-sync/internal/github"
	"gha-file-sync/internal/log"
	"gha-file-sync/internal/sync"
)

func main() {
	ctx := context.Background()

	// init of logger
	log.Init()

	// init of config
	log.Infof("Configuration...")
	config, err := cfg.InitConfig()
	if err != nil {
		log.Errorf("initing config: %v", err)
		os.Exit(1)
	}
	config.Print()

	// init of github client
	ghClient, err := github.NewClient(ctx, config.GithubToken)
	if err != nil {
		log.Errorf("initing github client: %v", err)
		os.Exit(1)
	}

	// start synchronization
	log.Infof("Let's sync")
	for _, repoName := range config.RepositoryNames {
		if err := sync.Do(ctx, repoName, config, ghClient); err != nil {
			log.Errorf("syncing %s: %v", repoName, err)
		}
	}
	log.Infof("Sync finished.")
}

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RepositoryNames []string
	FilesBindings   []string

	IsDryRun bool

	GithubToken string
	GithubURL   string

	CommitMessage        string
	FileSyncBranchRegexp string

	Workspace string
}

func initConfig() (c Config, err error) {
	if c.RepositoryNames, err = getRepositoryNames(); err != nil {
		return c, err
	}
	if c.FilesBindings, err = getFilesBindings(); err != nil {
		return c, err
	}
	if c.IsDryRun, err = getDryRun(); err != nil {
		return c, err
	}
	if c.GithubToken, err = getGithubToken(); err != nil {
		return c, err
	}
	if c.GithubURL, err = getGithubURL(); err != nil {
		return c, err
	}
	if c.CommitMessage, err = getCommitMessage(); err != nil {
		return c, err
	}
	if c.FileSyncBranchRegexp, err = getFileSyncBranchRegexp(); err != nil {
		return c, err
	}
	if c.Workspace, err = getWorkspace(); err != nil {
		return c, err
	}
	return c, nil
}

func getRepositoryNames() ([]string, error) {
	// get the raw list from env
	fmt.Println("repos:", os.Getenv("REPOSITORIES"))
	repoNamesStr := os.Getenv("REPOSITORIES")
	if repoNamesStr == "" {
		return nil, fmt.Errorf("REPOSITORIES is empty but required")
	}
	// trim spaces
	repoNamesStr = strings.TrimSpace(repoNamesStr)
	// split by \n
	repoNames := strings.Split(repoNamesStr, "\n")
	return repoNames, nil
}

func getFilesBindings() ([]string, error) {
	// get the raw list from env
	filesBindingsStr := os.Getenv("FILES_BINDINGS")
	if filesBindingsStr == "" {
		return nil, fmt.Errorf("FILES_BINDINGS is empty but required")
	}
	// trim spaces
	filesBindingsStr = strings.TrimSpace(filesBindingsStr)
	// split by \n
	fileBindings := strings.Split(filesBindingsStr, "\n")
	return fileBindings, nil
}

func getDryRun() (bool, error) {
	isDryRunStr := os.Getenv("DRY_RUN")
	// default is true
	if isDryRunStr == "" {
		log.Printf("DRY_RUN empty: set to default value")
		return true, nil
	}
	return strconv.ParseBool(isDryRunStr)
}

func getGithubToken() (string, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return "", fmt.Errorf("GITHUB_TOKEN is empty but required")
	}
	return githubToken, nil
}

func getGithubURL() (string, error) {
	githubURL := os.Getenv("GITHUB_URL")
	if githubURL == "" {
		return "", fmt.Errorf("GITHUB_URL is empty but required")
	}
	return githubURL, nil
}

func getCommitMessage() (string, error) {
	commitMessage := os.Getenv("COMMIT_MESSAGE")
	if commitMessage == "" {
		return "", fmt.Errorf("COMMIT_MESSAGE is empty but required")
	}
	// auto-truncate commit message - 80 characters maximum
	if len(commitMessage) > 80 {
		commitMessage = commitMessage[80:]
	}
	return commitMessage, nil
}

func getFileSyncBranchRegexp() (string, error) {
	fileSyncBranchRegexp := os.Getenv("FILE_SYNC_BRANCH_REGEXP")
	if fileSyncBranchRegexp == "" {
		return "", fmt.Errorf("FILE_SYNC_BRANCH_REGEXP is empty but required")
	}
	return fileSyncBranchRegexp, nil
}

func getWorkspace() (string, error) {
	workspace := os.Getenv("WORKSPACE")
	if workspace == "" {
		return "", fmt.Errorf("WORKSPACE is empty but required")
	}
	return workspace, nil
}

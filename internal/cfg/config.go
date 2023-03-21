package cfg

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"git-file-sync/internal/log"
)

type Config struct {
	RepositoryNames []string
	FilesBindings   map[string]string

	IsDryRun bool

	GithubToken string
	GithubURL   string

	CommitMessage        string
	FileSyncBranchRegexp string

	Workspace string
}

func InitConfig() (c Config, err error) {
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

func (c Config) Print() {
	log.Infof("Repositories names:")
	for _, rn := range c.RepositoryNames {
		log.Infof("\t%s", rn)
	}
	log.Infof("Files bindings:")
	for src, dest := range c.FilesBindings {
		log.Infof("\t%s -> %s", src, dest)
	}
	log.Infof("Dry Run: %v", c.IsDryRun)
	log.Infof("Is GitHub token empty? %v", (len(c.GithubToken) == 0))
	log.Infof("Github host URL: %v", c.GithubURL)
	log.Infof("Commit message: %v", c.CommitMessage)
	log.Infof("File sync branch regexp: %v", c.FileSyncBranchRegexp)
	log.Infof("Workspace: %v", c.Workspace)
}

func getRepositoryNames() ([]string, error) {
	// get the raw list from env
	repoNamesStr := os.Getenv("REPOSITORIES")
	if repoNamesStr == "" {
		return nil, fmt.Errorf("REPOSITORIES is empty but required")
	}
	// trim spaces
	repoNamesStr = strings.TrimSpace(repoNamesStr)
	// split by \n
	repoNames := strings.Split(repoNamesStr, "\n")

	for _, name := range repoNames {
		if len(strings.Split(name, "/")) != 2 {
			return nil, fmt.Errorf("invalid repo name: %s {OWNER}/{NAME} expected", name)
		}
	}
	return repoNames, nil
}

func getFilesBindings() (map[string]string, error) {
	// get the raw list from env
	filesBindingsStr := os.Getenv("FILES_BINDINGS")
	if filesBindingsStr == "" {
		return nil, fmt.Errorf("FILES_BINDINGS is empty but required")
	}
	// trim spaces
	filesBindingsStr = strings.TrimSpace(filesBindingsStr)
	// split by \n
	fileBindingsList := strings.Split(filesBindingsStr, "\n")

	filesBindings := make(map[string]string, len(fileBindingsList))
	// split each binding by `=` to build the binding map
	for _, fileBindingStr := range fileBindingsList {
		split := strings.Split(fileBindingStr, "=")
		if len(split) != 2 {
			return nil, fmt.Errorf("incorrect binding: %s", fileBindingStr)
		}
		filesBindings[split[0]] = split[1]
	}

	// error if no files bindings have been found
	if len(filesBindings) == 0 {
		return nil, fmt.Errorf("no valid files bindings found")
	}
	return filesBindings, nil
}

func getDryRun() (bool, error) {
	isDryRunStr := os.Getenv("DRY_RUN")
	// default is true
	if isDryRunStr == "" {
		log.Infof("DRY_RUN empty: set to default value")
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
	log.Infof("raw commit message: %s", commitMessage)
	// auto-truncate commit message - 120 characters maximum
	if len(commitMessage) > 120 {
		commitMessage = commitMessage[:119]
	}
	log.Infof("final commit message: %s", commitMessage)
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

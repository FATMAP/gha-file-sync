package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"

	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
)

type RepoManager struct {
	RepoName string

	localPath            string
	githubHostURL        string
	githubToken          string
	fileSyncBranchRegexp *regexp.Regexp

	repository          *git.Repository
	branchToCompareWith *gitconfig.Branch
	updateMode          bool
}

func NewRepoManager(repoName, baseLocalPath, githubURL, githubToken, fileSyncBranchRegexpStr string) RepoManager {
	rm := RepoManager{
		RepoName:      repoName,
		localPath:     path.Join(baseLocalPath, repoName),
		githubHostURL: githubURL,
		githubToken:   githubToken,
	}
	rm.fileSyncBranchRegexp = regexp.MustCompile(fileSyncBranchRegexpStr)
	return rm
}

func (rm *RepoManager) buildRepoURL() string {
	return fmt.Sprintf("https://x-access-token:%s@%s/%s.git", rm.githubToken, rm.githubHostURL, rm.RepoName)
}

func (rm *RepoManager) Clone(ctx context.Context) error {
	authorizedRepoURL := rm.buildRepoURL()

	r, err := git.PlainCloneContext(ctx, rm.localPath, false, &git.CloneOptions{
		URL:      authorizedRepoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return err
	}
	rm.repository = r
	return nil
}

func (rm *RepoManager) PickBranchToCompare() error {
	cfg, err := rm.repository.Config()
	if err != nil {
		return fmt.Errorf("retrieving config: %v", err)
	}

	// try to find an existing file sync pr
	alreadyFound := false
	for name, branch := range cfg.Branches {
		if rm.fileSyncBranchRegexp.MatchString(name) {
			if alreadyFound { // this means we do existing file sync pr because it was already found previously
				log.Printf("WARN: it seems there are two existing file sync pull request")
				// TODO: take the latest one? close the oldest one?
			}
			rm.branchToCompareWith = branch
			rm.updateMode = true
			alreadyFound = true
		}
	}

	// if no existing pr was found, take either main or master branch
	var exist bool
	rm.branchToCompareWith, exist = cfg.Branches["main"]
	if !exist {
		rm.branchToCompareWith, exist = cfg.Branches["master"]
		if !exist {
			return fmt.Errorf("could not find any branch to compare with")
		}
	}
	return nil
}

// TODO
func (rm *RepoManager) GetDiff() (string, error) {
	// Files Bindings Logic
	// 1. copy from github-file-sync to local_path the files according to FILES_BINDINGS
	// where I am able to find the file of github-file-sync

	fmt.Println("---------------------------------")
	fmt.Println(rm.branchToCompareWith.Name)

	fmt.Println("pwd:")
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	fmt.Println(path)

	fmt.Println("lsdir")
	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		fmt.Println(f.Name())
	}

	// 2. Show the Diff
	// 2. do a git status --porcelain

	// 3.
	return "", nil
}

// TODO
func (rm *RepoManager) CreateOrUpdateFileSyncPR() error {
	if rm.updateMode {
		fmt.Println("updating existing file sync pull request")
	}
	return nil
}

func (rm *RepoManager) CleanAll() error {
	// remove all local files
	return os.RemoveAll(rm.localPath)
}

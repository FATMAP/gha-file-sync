package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	cp "github.com/otiai10/copy"
)

type RepoManager struct {
	repoName string
	owner    string

	localPath            string
	fileSyncBranchRegexp *regexp.Regexp

	ghHostURL string
	ghToken   string
	ghClient  Client

	repository *git.Repository
	headBranch string
	createMode bool

	fileBindings map[string]string
}

func NewRepoManager(
	repoName, owner, baseLocalPath, githubURL, githubToken, fileSyncBranchRegexpStr string,
	ghClient Client,
	fileBindings map[string]string,
) RepoManager {
	rm := RepoManager{
		repoName:     repoName,
		owner:        owner,
		localPath:    path.Join(baseLocalPath, owner, repoName),
		ghHostURL:    githubURL,
		ghToken:      githubToken,
		ghClient:     ghClient,
		fileBindings: fileBindings,
		createMode:   true, // by default, consider creating a new PR
	}

	rm.fileSyncBranchRegexp = regexp.MustCompile(fileSyncBranchRegexpStr)
	return rm
}

func (rm *RepoManager) buildRepoURL() string {
	return fmt.Sprintf("https://x-access-token:%s@%s/%s/%s.git", rm.ghToken, rm.ghHostURL, rm.owner, rm.repoName)
}

func (rm *RepoManager) Clone(ctx context.Context) error {
	authorizedRepoURL := rm.buildRepoURL()

	r, err := git.PlainCloneContext(
		ctx, rm.localPath, false,
		&git.CloneOptions{URL: authorizedRepoURL},
	)
	if err != nil {
		return err
	}
	rm.repository = r
	return nil
}

// HasChangedAfterCopy first syncs locally bound files then checks status and returns true if something has changed
func (rm *RepoManager) HasChangedAfterCopy(ctx context.Context) (bool, error) {
	// return directly if no files bindings defined
	if len(rm.fileBindings) == 0 {
		return false, nil
	}

	// 1. copy files from the current repository to the repo-to-sync local path
	// according to configured bindings
	atLeastOneSuccess := false
	for src, dest := range rm.fileBindings {
		if err := cp.Copy(src, path.Join(rm.localPath, dest)); err != nil {
			log.Printf("ERROR: copying %s to %s: %v", src, dest, err)
			continue
		}
		atLeastOneSuccess = true
	}
	if !atLeastOneSuccess {
		return false, fmt.Errorf("not able to copy any file")
	}

	// 2. consider if files have changed / being created by running the git status command
	workTree, err := rm.repository.Worktree()
	if err != nil {
		return false, fmt.Errorf("getting worktree: %v", err)
	}
	statuses, err := workTree.Status()
	if err != nil {
		return false, fmt.Errorf("getting status: %v", err)
	}
	// return true of status return a non empty result
	return (len(statuses) > 0), nil
}

// SetHeadBranch on the repo to sync to see if something has changed
// could be:
// - the repo's HEAD: main or master probably
// - an existing file sync branch, which set the finalBaseBranch if found
func (rm *RepoManager) SetHeadBranch(ctx context.Context) error {
	// try to find an existing file sync branch by checkout opened PRs
	branchNames, err := rm.ghClient.GetBranchNamesFromPRs(ctx, rm.owner, rm.repoName)
	if err != nil {
		return fmt.Errorf("getting branches: %v", err)
	}

	// try to find an existing file sync pr
	alreadyFound := false
	for _, name := range branchNames {
		if rm.fileSyncBranchRegexp.MatchString(name) {
			if alreadyFound { // this means we do existing file sync pr because it was already found previously
				log.Printf("WARN: it seems there are two existing file sync pull request")
				// TODO: take the latest one? close the oldest one?
			}
			rm.headBranch = name
			rm.createMode = false

			alreadyFound = true
		}
	}

	if rm.createMode {
		rm.headBranch = fmt.Sprintf("%d-%d-%d-sync-file-pr", time.Now().Year(), time.Now().Month(), time.Now().Day())
	}
	return nil
}

// CreateOrUpdateFileSyncPR
func (rm *RepoManager) UpdateHeadBranch(ctx context.Context) error {
	// move to the repository
	if err := os.Chdir(rm.localPath); err != nil {
		return fmt.Errorf("moving to local path: %v", err)
	}

	// checkout the head branch
	workTree, err := rm.repository.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %v", err)
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Keep: true, // keep actual changes - files already copied

		Branch: plumbing.ReferenceName(rm.headBranch), // according to existing pr
		Create: rm.createMode,                         // according to existing pr
	})
	if err != nil {
		return fmt.Errorf("checking out: %v", err)
	}

	if err := workTree.AddGlob("."); err != nil {
		return fmt.Errorf("adding: %v", err)
	}

	// commit
	if err := workTree.Commit("")

	// push

	// open PR
	fmt.Println("non update mode")
	return nil
}

func (rm *RepoManager) CleanAll(ctx context.Context) error {
	// remove all local files
	return os.RemoveAll(rm.localPath)
}

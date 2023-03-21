package github

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"time"

	"git-file-sync/internal/log"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

	repository   *git.Repository
	baseBranch   string
	createPRMode bool

	fileBindings map[string]string
}

func NewRepoManager(
	repoName, owner, baseLocalPath, githubURL, githubToken, fileSyncBranchRegexpStr string,
	ghClient Client,
	fileBindings map[string]string,
) RepoManager {
	rm := RepoManager{
		repoName: repoName,
		owner:    owner,

		localPath: path.Join(baseLocalPath, owner, repoName),

		ghHostURL: githubURL,
		ghToken:   githubToken,
		ghClient:  ghClient,

		baseBranch:   fmt.Sprintf("%s-sync-file-pr", time.Now().Format("2006-01-02")), // default branch
		createPRMode: true,                                                            // by default, consider creating a new PR

		fileBindings: fileBindings,
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

// PickBaseBranch on the repo manager structure which will be used to compare files
// could be:
// - a new branch based on the repo's HEAD: probably main or master
// - an existing file sync branch
func (rm *RepoManager) PickBaseBranch(ctx context.Context) error {
	// try to find an existing file sync branch by checking opened PRs
	branchNames, err := rm.ghClient.GetBranchNamesFromPRs(ctx, rm.owner, rm.repoName)
	if err != nil {
		return fmt.Errorf("getting branches: %v", err)
	}

	// try to find an existing file sync PR
	alreadyFound := false
	for _, name := range branchNames {
		log.Infof("branch name: %s", name)
		// use branch name to see if it is an file sync PR
		if rm.fileSyncBranchRegexp.MatchString(name) {
			if alreadyFound {
				log.Warnf("it seems there are two existing file sync pull request on repo %s", rm.repoName)
				// TODO: take the latest one? close the oldest one?
			}
			rm.baseBranch = name
			rm.createPRMode = false

			alreadyFound = true
		}
	}
	log.Infof("final base branch is %s", rm.baseBranch)
	log.Infof("candidates: %v", branchNames)
	log.Infof("found? %v", alreadyFound)
	return nil
}

// HasChangedAfterCopy first update locally files following binding rules
// then checks the git status and returns true if something has changed
func (rm *RepoManager) HasChangedAfterCopy(ctx context.Context) (bool, error) {
	// return directly if no files bindings defined
	if len(rm.fileBindings) == 0 {
		return false, nil
	}

	// baseBranch should be set
	if rm.baseBranch == "" {
		return false, fmt.Errorf("baseBranch is not set")
	}

	// 1. checkout the base branch to compare
	workTree, err := rm.repository.Worktree()
	if err != nil {
		return false, fmt.Errorf("getting worktree: %v", err)
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Keep: true, // keep actual local changes

		Branch: plumbing.ReferenceName(rm.baseBranch),
		Create: rm.createPRMode,
	})
	if err != nil {
		return false, fmt.Errorf("checking out: %v", err)
	}

	// 2. copy files from the current repository to the repo-to-sync local path
	// according to configured bindings
	atLeastOneSuccess := false
	for src, dest := range rm.fileBindings {
		if err := cp.Copy(src, path.Join(rm.localPath, dest)); err != nil {
			log.Errorf("copying %s to %s: %v", src, dest, err)
			continue
		}
		atLeastOneSuccess = true
	}
	if !atLeastOneSuccess {
		return false, fmt.Errorf("not able to copy any file")
	}

	// 3. consider if files have changed / being created by running the git status command
	statuses, err := workTree.Status()
	if err != nil {
		return false, fmt.Errorf("getting status: %v", err)
	}
	// return true of status return a non empty result
	return (len(statuses) > 0), nil
}

func (rm *RepoManager) UpdateRemote(ctx context.Context, commitMsg string) error {
	// move to the repository
	if err := os.Chdir(rm.localPath); err != nil {
		return fmt.Errorf("moving to local path: %v", err)
	}

	// checkout the base branch to update
	workTree, err := rm.repository.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %v", err)
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Keep: true, // keep actual changes

		Branch: plumbing.ReferenceName(rm.baseBranch), // according to existing pr
		Create: false,                                 // already created if necessary in HasChangedAfterCopy
	})
	if err != nil {
		return fmt.Errorf("checking out: %v", err)
	}

	// add all files
	if err := workTree.AddGlob("."); err != nil {
		return fmt.Errorf("adding: %v", err)
	}

	// commit changes
	commitOpt := &git.CommitOptions{
		All: true, // TODO: to test new added file
		Author: &object.Signature{
			Name:  "FATMAPRobot",
			Email: "robots@fatmap.com",
			When:  time.Now(),
		},
	}
	if _, err := workTree.Commit(commitMsg, commitOpt); err != nil {
		return fmt.Errorf("commiting: %v", err)
	}

	// push to remote
	pushOpt := &git.PushOptions{
		RemoteName:     rm.baseBranch,
		Force:          true,
		ForceWithLease: &git.ForceWithLease{RefName: plumbing.ReferenceName(rm.baseBranch)},
		Atomic:         true,
	}
	if err := rm.repository.PushContext(ctx, pushOpt); err != nil {
		return fmt.Errorf("pushing: %v", err)
	}

	// TODO: open or update PR
	return nil
}

func (rm *RepoManager) CleanAll(ctx context.Context) error {
	// remove all local files
	return os.RemoveAll(rm.localPath)
}

package git

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Repository struct {
	// git config
	syncBranchName string
	repoURL        string
	localPath      string

	// auth config
	auth       *http.BasicAuth
	authorName string

	// internal state
	baseBranchName string // main or master
	repo           *git.Repository
	workTree       *git.Worktree
	syncRef        *plumbing.Reference
}

// NewRepository clones a repository locally based on given parameters and returns a reference to its object
func NewRepository(
	ctx context.Context,
	localPath, repoURL, syncBranchName string,
	auth *http.BasicAuth, authorName string,
) (*Repository, error) {
	// init the repository
	r := &Repository{
		localPath:      localPath,
		repoURL:        repoURL,
		syncBranchName: syncBranchName,

		auth:       auth,
		authorName: authorName,
	}

	// try to clone to local path
	opt := &git.CloneOptions{
		URL:  r.repoURL,
		Auth: r.auth,
	}
	isBare := false
	repo, err := git.PlainCloneContext(ctx, localPath, isBare, opt)
	if err != nil {
		return nil, fmt.Errorf("cloning: %v", err)
	}
	r.repo = repo

	return r, nil
}

// GetBaseBranchName based on the repo
// first it tries to get a "main" branch, then a "master" branch, then it fails
func (r *Repository) GetBaseBranchName() (string, error) {
	if r.baseBranchName == "" {
		c, err := r.repo.Config()
		if err != nil {
			return "", fmt.Errorf("could not get config: %v", err)
		}
		r.baseBranchName = "main" // consider naively that it is "main"

		if _, ok := c.Branches["main"]; !ok {
			if _, ok := c.Branches["master"]; !ok {
				return "", fmt.Errorf("no base branch found (main|master)")
			}
			r.baseBranchName = "master"
		}
	}
	return r.baseBranchName, nil
}

// GetSyncBranchName
func (r *Repository) GetSyncBranchName() string {
	return r.syncBranchName
}

// SetSyncBranchName
func (r *Repository) SetSyncBranchName(name string) {
	r.syncBranchName = name
}

// IsNotSetup returns true if any important internal state variable is not set
func (r *Repository) IsNotSetup() bool {
	return (r.repo == nil ||
		r.workTree == nil ||
		r.syncRef == nil)
}

// Add, Commit, Push from the current local folder to remote
func (r *Repository) AddCommitPush(
	ctx context.Context, commitMsg string,
) error {
	// add all files
	opt := git.AddOptions{
		All:  true,
		Path: r.localPath,
	}
	if err := r.workTree.AddWithOptions(&opt); err != nil {
		return fmt.Errorf("adding: %v", err)
	}

	// commit changes
	commitOpt := &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name: r.authorName,
			When: time.Now(),
		},
	}
	if _, err := r.workTree.Commit(commitMsg, commitOpt); err != nil {
		return fmt.Errorf("commiting: %v", err)
	}

	// push to remote
	pushOpt := &git.PushOptions{
		RemoteName: "origin",
		Auth:       r.auth,
		Force:      true,
		Atomic:     true,
	}
	if err := r.repo.PushContext(ctx, pushOpt); err != nil {
		return fmt.Errorf("pushing: %v", err)
	}
	return nil
}

// ChangesDetected returns true if the git status command returns elements
func (r *Repository) ChangeDetected() (bool, error) {
	statuses, err := r.workTree.Status()
	if err != nil {
		return false, fmt.Errorf("getting status: %v", err)
	}
	// return true of status return a non empty result
	return (len(statuses) > 0), nil
}

func (r *Repository) Clean() error {
	return os.RemoveAll(r.localPath)
}

// SetupLocalSyncBranch performs low level git operations to setup sync branch
// it handles it either a remote branch already exist or if it should be created
func (r *Repository) SetupLocalSyncBranch(isNewBranch bool) error {
	var err error
	branchConfig := &config.Branch{
		Name:   r.syncBranchName,
		Rebase: "true",
	}
	// a. new branch mode: symbolic ref and branch merge ref are based on the current local head ref
	// b. existing branch mode: symbolic ref and branch merge ref are based on the existing remote ref
	if isNewBranch { // a
		headRef, err := r.repo.Head()
		if err != nil {
			return fmt.Errorf("getting head: %v", err)
		}
		r.syncRef = plumbing.NewSymbolicReference(plumbing.NewBranchReferenceName(r.syncBranchName), headRef.Name())
		branchConfig.Merge = r.syncRef.Name()
	} else { // b
		r.syncRef = plumbing.NewSymbolicReference(plumbing.NewBranchReferenceName(r.syncBranchName), plumbing.NewRemoteReferenceName("origin", r.syncBranchName))
		branchConfig.Merge = r.syncRef.Name()
		branchConfig.Remote = r.syncBranchName
	}

	// set the local storer reference
	if err := r.repo.Storer.SetReference(r.syncRef); err != nil {
		return fmt.Errorf("setting final ref: %v", err)
	}
	// init the work tree
	r.workTree, err = r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %v", err)
	}
	// create the branch reference locally - set the merge to the recently created ref
	if err := r.repo.CreateBranch(branchConfig); err != nil {
		return fmt.Errorf("creating remote branch: %v", err)
	}
	// checkout the sync ref in the work tree
	co := &git.CheckoutOptions{Branch: r.syncRef.Name()}
	if err := r.workTree.Checkout(co); err != nil {
		return fmt.Errorf("checkout %s: %v", r.syncRef.String(), err)
	}
	return nil
}

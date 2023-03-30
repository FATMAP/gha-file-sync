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
	repo     *git.Repository
	workTree *git.Worktree
	syncRef  *plumbing.Reference
}

// NewRepository clones a repository locally based on given parameters and returns a reference to its object
// /!\ it is not concurrent safe
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
func (r *Repository) GetBaseBranchName() string {
	return "main" // TODO: handle master
}

// GetSyncBranchName
func (r *Repository) GetSyncBranchName() string {
	return r.syncBranchName
}

// SetSyncBranchName
func (r *Repository) SetSyncBranchName(name string) {
	r.syncBranchName = name
}

// IsSetup returns true if all internal state has been set
func (r *Repository) IsSetup() bool {
	return (r.repo != nil &&
		r.workTree != nil &&
		r.syncRef != nil)
}

// Add, Commit, Push from the current local folder to remote
func (r *Repository) AddCommitPush(
	ctx context.Context, commitMsg string,
) error {
	// move to the repo
	// TODO: make this asynchronously callable by using AddOption and
	if err := os.Chdir(r.localPath); err != nil {
		return fmt.Errorf("moving to local path: %v", err)
	}

	// add all files
	if err := r.workTree.AddGlob("."); err != nil {
		return fmt.Errorf("adding: %v", err)
	}

	// commit changes
	commitOpt := &git.CommitOptions{
		All: true, // TODO test with false + deleted files
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

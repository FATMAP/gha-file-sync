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
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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

	repo       *git.Repository
	workTree   *git.Worktree
	syncBranch string
	syncRef    *plumbing.Reference
	// existingPRNumber is used for PR update and also indicates if PR and branch should be created - no distinction between these two elements for now
	// it is set based on PR first (if sync branch exists without, it is either ignored or results in an error)
	existingPRNumber *int

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

		syncBranch:       fmt.Sprintf("%s-sync-file-pr", time.Now().Format("2006-01-02")), // default branch
		existingPRNumber: nil,                                                             // by default, consider creating a new PR

		fileBindings: fileBindings,
	}

	rm.fileSyncBranchRegexp = regexp.MustCompile(fileSyncBranchRegexpStr)
	return rm
}

func (rm *RepoManager) getRepoURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", rm.ghHostURL, rm.owner, rm.repoName)
}

func (rm *RepoManager) getBasicAuth() *http.BasicAuth {
	return &http.BasicAuth{Username: rm.ghToken, Password: "x-oauth-basic"}
}

func (rm *RepoManager) Clone(ctx context.Context) error {
	fmt.Println("local path for repo is: ", rm.localPath)
	os.RemoveAll(rm.localPath)
	// TODO: remove 2 lines above
	r, err := git.PlainCloneContext(
		ctx, rm.localPath, false,
		&git.CloneOptions{
			URL:  rm.getRepoURL(),
			Auth: rm.getBasicAuth(),
		},
	)
	if err != nil {
		return err
	}
	rm.repo = r
	return nil
}

// PicksyncBranch on the repo manager structure which will be used to compare files
// could be:
// - a new branch based on the repo's HEAD: probably main or master
// - an existing file sync branch
func (rm *RepoManager) PickSyncBranch(ctx context.Context) error {
	// try to find an existing file sync branch by checking opened PRs
	branchNameByPRNumbers, err := rm.ghClient.GetBranchNameByPRNumbers(ctx, rm.owner, rm.repoName)
	if err != nil {
		return fmt.Errorf("getting branches: %v", err)
	}

	// try to find an existing file sync PR
	alreadyFound := false
	for prNumber, branchName := range branchNameByPRNumbers {
		// use branch name to see if it is an file sync PR
		if rm.fileSyncBranchRegexp.MatchString(branchName) {
			if alreadyFound {
				log.Warnf("it seems there are two existing file sync pull request on repo %s", rm.repoName)
				// TODO: take the latest one? close the oldest one?
			}
			rm.syncBranch = branchName
			rm.existingPRNumber = &prNumber
			alreadyFound = true
		}
	}

	// configure branch locally
	if err := rm.setupLocalSyncBranch(); err != nil {
		return fmt.Errorf("setting up sync branch locally: %v", err)
	}
	return nil
}

// setupLocalSyncBranch performs low level git operation to setup sync branc
// it handles it either a remote branch already exist or if it should be created
func (rm *RepoManager) setupLocalSyncBranch() error {
	var err error
	branchConfig := &config.Branch{
		Name:   rm.syncBranch,
		Rebase: "true",
	}
	// a. new branch mode: symbolic ref and branch merge ref are based on the current local head ref
	// b. existing branch mode: symbolic ref and branch merge ref are based on the existing remote ref
	if rm.existingPRNumber == nil { // a
		headRef, err := rm.repo.Head()
		if err != nil {
			return fmt.Errorf("getting head: %v", err)
		}
		rm.syncRef = plumbing.NewSymbolicReference(plumbing.NewBranchReferenceName(rm.syncBranch), headRef.Name())
		branchConfig.Merge = rm.syncRef.Name()
	} else { // b
		rm.syncRef = plumbing.NewSymbolicReference(plumbing.NewBranchReferenceName(rm.syncBranch), plumbing.NewRemoteReferenceName("origin", rm.syncBranch))
		branchConfig.Merge = rm.syncRef.Name()
		branchConfig.Remote = rm.syncBranch
	}

	// set the local storer reference
	if err := rm.repo.Storer.SetReference(rm.syncRef); err != nil {
		return fmt.Errorf("setting final ref: %v", err)
	}
	// init the work tree
	rm.workTree, err = rm.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %v", err)
	}
	// create the branch reference locally - set the merge to the recently created ref
	if err := rm.repo.CreateBranch(branchConfig); err != nil {
		return fmt.Errorf("creating remote branch: %v", err)
	}
	// checkout the sync ref in the work tree
	co := &git.CheckoutOptions{Branch: rm.syncRef.Name()}
	if err := rm.workTree.Checkout(co); err != nil {
		return fmt.Errorf("checkout %s: %v", rm.syncRef.String(), err)
	}
	return nil
}

// HasChangedAfterCopy first update locally files following binding rules
// then checks the git status and returns true if something has changed
func (rm *RepoManager) HasChangedAfterCopy(ctx context.Context) (bool, error) {
	// return directly if no files bindings defined
	if len(rm.fileBindings) == 0 {
		return false, nil
	}

	// syncBranch and workTree should be set
	if rm.syncBranch == "" || rm.workTree == nil {
		return false, fmt.Errorf("syncBranch or workTree is not set")
	}

	// 2. copy files from the current repo to the repo-to-sync local path
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
	statuses, err := rm.workTree.Status()
	if err != nil {
		return false, fmt.Errorf("getting status: %v", err)
	}
	fmt.Println(statuses)
	// return true of status return a non empty result
	return (len(statuses) > 0), nil
}

func (rm *RepoManager) UpdateRemote(ctx context.Context, commitMsg, prTitle string) error {
	// move to the repo
	if err := os.Chdir(rm.localPath); err != nil {
		return fmt.Errorf("moving to local path: %v", err)
	}

	rm.printStatus(ctx, "before add ----")

	// add all files
	if err := rm.workTree.AddGlob("."); err != nil {
		return fmt.Errorf("adding: %v", err)
	}

	rm.printStatus(ctx, "before commit ----")

	// commit changes
	commitOpt := &git.CommitOptions{
		All: true, // TODO: to test new added file
	}
	if _, err := rm.workTree.Commit(commitMsg, commitOpt); err != nil {
		return fmt.Errorf("commiting: %v", err)
	}

	// push to remote
	pushOpt := &git.PushOptions{
		RemoteName: "origin",
		Auth:       rm.getBasicAuth(),
		Force:      true,
		Atomic:     true,
		// ForceWithLease: &git.ForceWithLease{RefName: plumbing.ReferenceName(rm.syncBranch)},
	}
	if err := rm.repo.PushContext(ctx, pushOpt); err != nil {
		return fmt.Errorf("pushing: %v", err)
	}

	if err := rm.ghClient.CreateOrUpdatePR(
		ctx, rm.existingPRNumber,
		rm.owner, rm.repoName,
		"main", rm.syncBranch,
		prTitle, commitMsg,
	); err != nil {
		return fmt.Errorf("creating/updating PR: %v", err)
	}
	return nil
}

func (rm *RepoManager) CleanAll(ctx context.Context) error {
	// remove the repository folder on local filesystem
	// return os.RemoveAll(rm.localPath)
	// TODO: uncomment remove of local path
	return nil
}

// printStatus is only used for debug purpose
func (rm *RepoManager) printStatus(_ context.Context, msg string) {
	fmt.Println(msg)
	statuses, err := rm.workTree.Status()
	if err != nil {
		log.Errorf("getting status: %v", err)
	}
	for k, v := range statuses {
		fmt.Printf("\t%v: staging '%s' vs worktree '%s'\n", k, string(v.Staging), string(v.Worktree))
	}
	if len(statuses) == 0 {
		fmt.Printf("\tno changes detected..")
	}
}

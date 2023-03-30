package sync

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"time"

	mygit "github-file-sync/internal/git"
	"github-file-sync/internal/github"
	"github-file-sync/internal/log"

	cp "github.com/otiai10/copy"
)

// Task is a handler which synchronize a git repository files on github with current filesystem and given rules called file bindings
type Task struct {
	// repo config
	repoName string
	owner    string

	// local tmp file config
	sourcePath string
	targetPath string

	// github config
	ghHostURL string
	ghToken   string
	ghClient  github.Client

	// git config
	gitRepo *mygit.Repository

	// additional config
	fileSyncBranchRegexp *regexp.Regexp
	fileBindings         map[string]string

	// internal state

	// existingPRNumber is used for PR update and also indicates if PR and branch should be created - no distinction between these two elements for now
	// it is set based on PR first (if sync branch exists without, it is either ignored or results in an error)
	existingPRNumber *int
}

// NewTask configured with default values and given parameters
// one task per repository
func NewTask(
	ctx context.Context,
	owner, repoName,
	baseSourcePath, baseTargetPath,
	ghURL, ghToken string,
	ghClient github.Client,
	fileSyncBranchRegexpStr string,
	fileBindings map[string]string,
) (t Task, err error) {
	// init the repo RepositoryManager
	t = Task{
		repoName: repoName,
		owner:    owner,

		sourcePath: baseSourcePath,
		targetPath: path.Join(baseTargetPath, owner, repoName),

		ghHostURL: ghURL,
		ghToken:   ghToken,
		ghClient:  ghClient,

		fileSyncBranchRegexp: regexp.MustCompile(fileSyncBranchRegexpStr),
		fileBindings:         fileBindings,

		existingPRNumber: nil, // by default, consider creating a new PR
	}

	// add to the repo RepositoryManager the author information
	authorName, err := ghClient.GetAuthenticatedUsername(ctx)
	if err != nil {
		return t, err
	}
	defaultBranchName := fmt.Sprintf("%s-sync-file-pr", time.Now().Format("2006-01-02"))
	t.gitRepo, err = mygit.NewRepository(
		ctx,
		t.targetPath,
		github.GetRepoURL(t.ghHostURL, t.owner, t.repoName), defaultBranchName,
		github.GetBasicAuth(t.ghToken), authorName,
	)
	if err != nil {
		return t, err
	}
	return t, nil
}

// PickSyncBranch on the repo which will be used to compare files  and push potential changes
// could be:
// - a new branch based on the repo's HEAD: probably main or master
// - an existing file sync branch
func (t *Task) PickSyncBranch(ctx context.Context) error {
	// try to find an existing file sync branch by checking opened PRs
	branchNameByPRNumbers, err := t.ghClient.GetHeadBranchNameByPRNumbers(ctx, t.owner, t.repoName)
	if err != nil {
		return fmt.Errorf("getting branches: %v", err)
	}

	// try to find an existing file sync PR
	alreadyFound := false
	for prNumber, branchName := range branchNameByPRNumbers {
		// use branch name to see if it is an file sync PR
		if t.fileSyncBranchRegexp.MatchString(branchName) {
			if alreadyFound {
				log.Warnf("it seems there are two existing file sync pull request on repo %s", t.repoName)
				// TODO: take the latest one? close the oldest one?
				break
			}
			alreadyFound = true
			t.gitRepo.SetSyncBranchName(branchName)

			t.existingPRNumber = new(int)
			*t.existingPRNumber = prNumber
		}
	}

	// configure the branch locally
	isNewBranch := (t.existingPRNumber == nil)
	if err := t.gitRepo.SetupLocalSyncBranch(isNewBranch); err != nil {
		return fmt.Errorf("setting up sync branch locally: %v", err)
	}
	return nil
}

// HasChangedAfterCopy first update locally files following binding rules
// then checks the git status and returns true if something has changed
func (t *Task) HasChangedAfterCopy(ctx context.Context) (bool, error) {
	// return directly if no files bindings defined
	if len(t.fileBindings) == 0 {
		return false, nil
	}

	// local git repo should be setup
	if t.gitRepo.IsNotSetup() {
		return false, fmt.Errorf("local git repo is not setup correctly")
	}

	// 2. copy files from the current repo to the repo-to-sync local path
	// according to configured bindings
	atLeastOneSuccess := false
	for src, dest := range t.fileBindings {
		// build absolute path to copy
		src = path.Join(t.sourcePath, src)
		dest = path.Join(t.targetPath, dest)

		if err := cp.Copy(src, dest); err != nil {
			log.Errorf("copying %s to %s: %v", src, dest, err)
			continue
		}

		atLeastOneSuccess = true
	}
	if !atLeastOneSuccess {
		return false, fmt.Errorf("not able to copy any file")
	}

	// 3. consider if files have changed
	return t.gitRepo.ChangeDetected()
}

func (t *Task) UpdateRemote(ctx context.Context, commitMsg, prTitle string) error {
	if err := t.gitRepo.AddCommitPush(ctx, commitMsg); err != nil {
		return err
	}
	baseBranchName, err := t.gitRepo.GetBaseBranchName()
	if err != nil {
		return err
	}
	if err := t.ghClient.CreateOrUpdatePR(
		ctx, t.existingPRNumber,
		t.owner, t.repoName,
		baseBranchName, t.gitRepo.GetSyncBranchName(),
		prTitle, commitMsg,
	); err != nil {
		return err
	}
	return nil
}

func (t *Task) CleanAll(ctx context.Context) error {
	return t.gitRepo.Clean()
}

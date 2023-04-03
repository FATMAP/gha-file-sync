# gha-file-sync

Github Action for Cross-Repo Files Synchronization using Pull Requests.
## What 

For a list of given repositories and file bindings, this action will open/update pull requests to synchronize the files that have changed.
The source of files is the repository where the action runs.

For each targeted repository:
  1. Clone the repository.
  2. Compute the final branch name and PR according to existing opened PRs.
  3. Check if changes have been made following files bindings configuration.
  4. Create or update a pull request if changes have been detected.
  5. Clean all created files locally.

## Configuration

See `action.yml` for more information.
## Missing

This action only manages to synchronize new files and updated files.
Removals or renames are not handled yet.

## Features wished to be added


#### Handling of removed/renamed files

Today, because only a raw copy of the source files is used, the removals and renames of files are not handled yet.
#### Commits messages as PR desc

Today, the PR description and comments are pointing by default to the release which triggered the synchronization.
It would be better to provide more information about the release, for example:
- the release description.
- the list of added commits.
- the original PR description.
#### 'Customs' Detection

Sometimes the synchronized files are customized locally for some reasons, it is hard to know about it when tens of repositories are involved.
The action should raise a warning somewhere if it detects a customization. Some rules as examples:

In order to find customization, it should compare the target files with the version `n - 1` of the source files to see if it was already differing.

- if it is a PR creation:
    - WARN in the PR desc
- if it is a PR update and the title does not contain `CUSTOM_DETECTED`:
  - WARN in a comment + update the title with sync `CUSTOM_DETECTED`
- if it is a PR update and the title contains `CUSTOM_DETECTED`:
  - WARN in a comment

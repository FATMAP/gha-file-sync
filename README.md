# gha-file-sync

Github Action for Cross-Repos File Synchronisation using Pull Requests.

## Configuration

See `action.yml` for more information.

## What 

For a list of given repositories and file bindings, this action will open/update pull requests to synchronize the files.
The files source is the repository where the action runs.

For each target repository:
  - Clone the repository.
  - Compute the final bra6nch and PR according to existing opened PRs.
  - Check if changes have been made following files bindings configuration.
  - Create or Update the pull request if something has changed.
  - Clean all created files locally.

## Missing

This action only manage to synchronize new files and updated files.
Remove or rename is not handled as of today.

## Features wished to be added

#### Commits messages as PR desc

Today, the PR desc and comments are pointing by default to the releases which triggered the synchronization.
It would be better to directly have the list of commits or even the original PR description.
#### Smart Custom Detection

Sometimes the synched files are customized locally, it should raise a warning when it is the case.

-  Warn about custom by checking with the previous version
    - if current_branch = main + custom detected
        - WARN in the PR desc
    - if current_branch = existing_pr + custom detected + no custom tag yet
      - WARN in a comment + update the title with sync CUSTOM
    - if current_branch = existing_pr + custom detected + custom tag detected
        - WARN in a comment

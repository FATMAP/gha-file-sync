# gha-file-sync

Github Action for Cross-Repos File Synchronisation using Pull Requests

## Features

**not implemented yet**

1. Get the list of repositories
2. For each repo:
  - Clone the repository
  - Get final branch - could be an existing file-sync pull request or main.
  - Make the diff following files bindings configuration
  - Create or Update the pull request if something has changed.
  - Remove the repository locally

#### Smart Custom Detection

**not implemented yet**

Sometimes the synched files are customized locally, we should raise warning when it is the case.

-  Warn about custom by checking with the previous version
    - if current_branch = main + custom detected
        - WARN in the PR desc
    - if current_branch = existing_pr + custom detected + no custom tag yet
      - WARN in a comment + update the title with sync CUSTOM
    - if current_branch = existing_pr + custom detected + custom tag detected
        - WARN in a comment

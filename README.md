# gha-file-sync

A Simple Github Action for Cross-Repo Files Synchronization using automatic Pull Requests.

<p align="center">
    <img src="assets/demo.gif"/>
</p>

<p align="center">
    <a href="#license">
        <img src="https://shields.io/badge/license-MIT-%23373737" />
    </a>
    <a href="https://godoc.org/github.com/FATMAP/gha-file-sync">
      <img src="https://godoc.org/github.com/FATMAP/gha-file-sync?status.svg" alt="GoDoc">
    </a>
</p>

## What 

For a list of given repositories and file bindings, this action will open/update pull requests to synchronize the files that have changed.
The source of files is the repository where the actual github action runs.

For each targeted repository:
  1. Clone the repository.
  2. Compute the final branch name and PR according to existing opened PRs.
  3. Check if changes have been made following files bindings configuration.
  4. Create or update a pull request if changes have been detected.
  5. Clean all created files locally.

## Configuration

See `action.yml` for more information about configuration
## Known issues

This action only manages to synchronize new files and updated files. Removals or renames are not handled yet.
:arrow_right: It is currently advised to blank a file that you want to remove to make it ineffective without having to remove it manually from all repositories.

## Potential Improvements

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

# Additional Information

## License

See the `LICENSE` file.

## Assets

<div>GIF made by 
  <a href="https://github.com/egonelbre/gophers" title="Egonelbre">@egonelbre</a> is licensed by <a href="https://creativecommons.org/publicdomain/zero/1.0/" title="Creative Commons BY 1.0" target="_blank">CC0 1.0</a>
</div>

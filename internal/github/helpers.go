package github

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func GetRepoURL(githubHostURL, repoOwner, repoName string) string {
	return fmt.Sprintf("https://%s/%s/%s.git", githubHostURL, repoOwner, repoName)
}

func GetBasicAuth(githubToken string) *http.BasicAuth {
	return &http.BasicAuth{Username: githubToken, Password: "x-oauth-basic"}
}

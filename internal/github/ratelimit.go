package github

import (
	"gha-file-sync/internal/log"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
)

func LogOnLimitDetected(ctx *github_ratelimit.CallbackContext) {
	log.Warnf("secondary rate limit detected, will continue at %v", ctx.SleepUntil)
}

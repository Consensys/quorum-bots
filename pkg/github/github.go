package github

type Github interface {
	GetGethReleaseData(tag string) ReleaseData
	GetGethTagComparison(base string, target string) TagCompare
	GetNextReleaseFrom(baseTag string) ReleaseData
	CreateQuorumPullRequest(branchName string, data ReleaseData, prBody string) PullRequestData
	FindOpenUpgradePullRequest(targetTag string) *PullRequestData
}
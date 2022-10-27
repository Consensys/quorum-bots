package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"upgradebot/config"
	"upgradebot/pkg/github"
)

const PullRequestTitleFormat = "[Upgrade] Go-Ethereum release %s"

type HTTPGithub struct {
	httpAdapter *HTTPClient
	config      *config.Config
}

func NewGithub(config *config.Config) github.Github {
	client := newHttpAdapter(config)
	return &HTTPGithub{
		httpAdapter: client,
		config:      config,
	}
}

// GetNextReleaseFrom - get the next go-ethereum release after a specific version/tag
func (api *HTTPGithub) GetNextReleaseFrom(baseTag string) github.ReleaseData {
	releases := api.GetAllGethReleases()
	releaseIndex := 0

	for i, r := range releases {
		if r.Tag == baseTag {
			releaseIndex = i - 1
		}
	}

	if releaseIndex < 0 {
		log.Fatal("Next release error notn found")
	}

	return releases[releaseIndex]
}

// GetAllGethReleases - get all go-ethereum releases
func (api *HTTPGithub) GetAllGethReleases() []github.ReleaseData {
	body, err := api.httpAdapter.DoGet(api.config.GethGithubAPIUrl + "/releases")
	if err != nil {
		log.Fatal(err)
	}
	var data []github.ReleaseData
	parseJson(body, &data)
	return data
}

// GetGethReleaseData - get go-ethereum release data based on a tag
func (api *HTTPGithub) GetGethReleaseData(tag string) github.ReleaseData {
	url := fmt.Sprintf("%s/releases/tags/%s", api.config.GethGithubAPIUrl, tag)

	body, err := api.httpAdapter.DoGet(url)
	if err != nil {
		log.Fatal(err)
	}
	data := github.ReleaseData{}
	parseJson(body, &data)

	return data
}

// GetGethTagComparison - compare two geth tags and extract PR merged and files changed
func (api *HTTPGithub) GetGethTagComparison(base string, target string) github.TagCompare {
	commitChanges := api.getCommitChanges(base, target)
	prsData := api.getPullRequests(commitChanges)
	return github.TagCompare{PullRequests: prsData, Files: commitChanges.Files}
}

// CreateQuorumPullRequest - create PR in the quorum repo
func (api *HTTPGithub) CreateQuorumPullRequest(branchName string, data github.ReleaseData, prBody string) (*github.PullRequestData, error) {
	title := fmt.Sprintf(PullRequestTitleFormat, data.Tag)
	createPrBody := github.CreatePullRequest{
		Title: title,
		Body:  prBody,
		Base:  "master",
		Head:  "quorumbot:" + branchName, // created from QuorumBot fork
		Draft: true,
	}

	jsonReader, err := newReader(createPrBody)
	if err != nil {
		return nil, fmt.Errorf("json reader: %w", err)
	}

	response, err := api.httpAdapter.DoPost(api.config.QuorumAPIUrl+"/pulls", jsonReader)
	if err != nil {
		return nil, fmt.Errorf("do post: %w", err)
	}

	result := &github.PullRequestData{}
	parseJson(response, result)

	return result, nil
}

// AddLabelsToIssue - adds some labels to the issue
func (api *HTTPGithub) AddLabelsToIssue(issueNumber int, labels ...string) *github.LabelsRequestData {
	// POST {{baseUrl}}/repos/:owner/:repo/issues/:issue_number/labels a JSON body labels -> array of strings
	labelsBody := github.LabelsRequest{Labels: labels}
	jsonReader, readerErr := newReader(labelsBody)
	if readerErr != nil {
		log.Fatal(readerErr)
		return nil
	}
	response, err := api.httpAdapter.DoPost(fmt.Sprintf("%s/issues/%d/labels", api.config.QuorumAPIUrl, issueNumber), jsonReader)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	result := &github.LabelsRequestData{}
	parseJson(response, result)

	return result
}

func (api *HTTPGithub) FindOpenUpgradePullRequest(targetTag string) *github.PullRequestData {
	title := fmt.Sprintf(PullRequestTitleFormat, targetTag)

	response, _ := api.httpAdapter.DoGet(api.config.QuorumAPIUrl + "/pulls?state=open&per_page=100")

	var result []github.PullRequestData
	parseJson(response, &result)

	for _, pr := range result {
		if pr.Title == title {
			return &pr
		}
	}
	return nil
}

func (api *HTTPGithub) getPullRequests(commitChanges github.CommitChanges) []github.PullRequest {
	prsData := api.getPullRequestDataFromCommits(commitChanges)

	pullRequests := make([]github.PullRequest, len(prsData))

	for i, prData := range prsData {
		files := api.getPullRequestFiles(prData)
		pr := github.PullRequest{Data: prData, Files: files}
		pullRequests[i] = pr
	}
	return pullRequests
}

func (api *HTTPGithub) getPullRequestFiles(prData github.PullRequestData) []github.File {
	url := fmt.Sprintf("%s/pulls/%d/files", api.config.GethGithubAPIUrl, prData.Number)

	body, _ := api.httpAdapter.DoGet(url)
	var prFiles []github.File
	parseJson(body, &prFiles)
	return prFiles
}

func (api *HTTPGithub) getPullRequestDataFromCommits(commitChanges github.CommitChanges) []github.PullRequestData {
	length := len(commitChanges.Commits)
	requestDataArray := make([]github.PullRequestData, 0)

	shas := make([]string, length)
	for i, c := range commitChanges.Commits {
		shas[i] = c.Sha[0:7]
	}

	for i := 0; i < length; i = i + 28 {
		end := math.Min(28, float64(length-i))
		requests := api.getPullRequestsData(shas[i : i+int(end)])
		requestDataArray = append(requestDataArray, requests...)
	}

	uniquePrs := make(map[int]bool)
	result := make([]github.PullRequestData, 0)

	// filter duplicates
	for _, pr := range requestDataArray {
		if uniquePrs[pr.Number] {
			continue
		}
		uniquePrs[pr.Number] = true
		result = append(result, pr)
	}

	return result
}

func (api *HTTPGithub) getPullRequestsData(shas []string) []github.PullRequestData {
	concatenatedSha := strings.Join(shas, "+")

	url := fmt.Sprintf("%s/search/issues?q=repo:ethereum/go-ethereum+is:pr+is:merged+merged+%s", api.config.GithubAPIUrl, concatenatedSha)
	body, err := api.httpAdapter.DoGet(url)
	if err != nil {
		log.Fatal(err)
	}

	prResult := struct {
		Items []github.PullRequestData
	}{}
	parseJson(body, &prResult)

	return prResult.Items
}

func (api *HTTPGithub) getCommitChanges(base string, target string) github.CommitChanges {
	url := fmt.Sprintf("%s/compare/%s...%s", api.config.GethGithubAPIUrl, base, target)
	body, _ := api.httpAdapter.DoGet(url)

	releaseCompare := github.CommitChanges{}
	parseJson(body, &releaseCompare)

	return releaseCompare
}

func parseJson(body []byte, data interface{}) {
	jsonErr := json.Unmarshal(body, data)
	if jsonErr != nil {
		log.Fatal(jsonErr, string(body))
	}
}

func newReader(data interface{}) (*bytes.Reader, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

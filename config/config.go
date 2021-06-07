package config

import (
	"os"
	"sync"
)

var once sync.Once

type Config struct {
	GithubAPIUrl string
	GithubLabel  string

	QuorumGitRepo string
	QuorumAPIUrl  string

	GethGitRepo      string
	GethGithubAPIUrl string

	GithubUsername  string
	GithubUserToken string

	QuorumRepoFolder      string
	QuorumVersionFilePath string
}

var (
	instance *Config
)

func GetConfig() *Config {
	once.Do(func() {
		githubUsername := os.Getenv("GITHUB_USERNAME")
		githubToken := os.Getenv("GITHUB_USER_TOKEN")
		instance = &Config{
			GithubAPIUrl: "https://api.github.com",
			GithubLabel:  "geth upstream upgrade",

			GithubUsername:  githubUsername,
			GithubUserToken: githubToken,

			GethGitRepo:      "https://github.com/ethereum/go-ethereum.git",
			GethGithubAPIUrl: "https://api.github.com/repos/ethereum/go-ethereum",

			QuorumGitRepo: "https://" + githubUsername + ":" + githubToken + "@github.com/Consensys/quorum.git",
			QuorumAPIUrl:  "https://api.github.com/repos/Consensys/quorum",

			// For experimentation with the private Quorum repository
			//QuorumGitRepo: "git@github.com:ConsenSysQuorum/quorum.git",
			//QuorumAPIUrl:  "https://api.github.com/repos/ConsenSysQuorum/quorum",

			QuorumRepoFolder:      "tmp-quorum-repo",
			QuorumVersionFilePath: "/params/version.go",
		}

	})
	return instance
}

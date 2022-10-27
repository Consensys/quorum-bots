package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"upgradebot/config"
)

type Git struct {
	config *config.Config
}

func NewGit(config *config.Config) *Git {
	return &Git{config: config}
}

// CloneQuorumRepository - clone the repository of Quorum locally and add the go-ethereum remote as `geth`
func (s *Git) CloneQuorumRepository() {
	err := exec.Command("git", "clone", s.config.QuorumGitRepo, s.config.QuorumRepoFolder).Run()
	if err != nil {
		log.Fatal(err)
	}

	// add quorum bot fork
	s.executeGitCommandOnRepo("remote", "add", "quorumbot", s.config.QuorumBotGitRepo)

	// load geth tags
	s.executeGitCommandOnRepo("remote", "add", "geth", s.config.GethGitRepo)
	s.executeGitCommandOnRepo("fetch", "geth", "--tags")
}

// ClearQuorumRepository - delete the repository folder
func (s *Git) ClearQuorumRepository() {
	exec.Command("rm", "-rf", s.config.QuorumRepoFolder).Run()
}

// CreateBranchFromGethTag - create a branch from a geth tag and push the branch to the remote quorum
func (s *Git) CreateBranchFromGethTag(targetTag string, branchName string) {
	s.executeGitCommandOnRepo("checkout", "tags/"+targetTag, "-b", branchName)
	s.executeGitCommandOnRepo("push", "-u", "quorumbot", branchName)
}

/*
GetBaseGethTag - Get current version of go-ethereum merged into Quorum

params/version.go

VersionMajor = 1        // Major version component of the current release
VersionMinor = 9        // Minor version component of the current release
VersionPatch = 8        // Patch version component of the current release
VersionMeta  = "stable" // Version metadata to append to the version string
*/
func (s *Git) GetBaseGethTag() string {
	matcherMajor, _ := regexp.Compile(`VersionMajor = (\d+)`)
	matcherMinor, _ := regexp.Compile(`VersionMinor = (\d+)`)
	matcherPatch, _ := regexp.Compile(`VersionPatch = (\d+)`)
	out, err := os.ReadFile(s.config.QuorumRepoFolder + s.config.QuorumVersionFilePath)
	if err != nil {
		log.Fatal("Error reading file "+s.config.QuorumVersionFilePath, err)
	}
	fileStr := string(out)

	if !matcherMajor.MatchString(fileStr) || !matcherMinor.MatchString(fileStr) || !matcherPatch.MatchString(fileStr) {
		log.Fatal("Failed to find the Geth version inside " + s.config.QuorumVersionFilePath)
	}

	majorVersion := matcherMajor.FindStringSubmatch(fileStr)[1]
	minorVersion := matcherMinor.FindStringSubmatch(fileStr)[1]
	patchVersion := matcherPatch.FindStringSubmatch(fileStr)[1]

	return fmt.Sprintf("v%s.%s.%s", majorVersion, minorVersion, patchVersion)
}

// GetConflictsFilesAgainstGethTargetVersion - Get the list of filenames that will have conflicts between Quorum master and the target geth tag
func (s *Git) GetConflictsFilesAgainstGethTargetVersion(targetGethTag string) []string {
	s.executeGitCommandOnRepo("merge", "--no-commit", "--no-ff", targetGethTag)
	defer s.executeGitCommandOnRepo("merge", "--abort")

	output, err := s.executeGitCommandOnRepo("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		log.Fatal(err)
	}

	return strings.Split(string(output), "\n")
}

// GetChangedFilesAgainstGethBaseVersion - Get the list of filenames that were changed by quorum when comparing with the same geth tag currently merged into quorum
func (s *Git) GetChangedFilesAgainstGethBaseVersion(baseGethTag string) []string {
	output, _ := s.executeGitCommandOnRepo("diff", "--name-only", baseGethTag)
	return strings.Split(string(output), "\n")
}

func (s *Git) executeGitCommandOnRepo(arg ...string) ([]byte, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = s.config.QuorumRepoFolder
	log.Println(cmd.String())
	return cmd.Output()
}

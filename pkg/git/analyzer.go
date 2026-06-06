package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/code-quality/cli/pkg/model"
)

type GitAnalyzer struct {
	repoPath string
}

func NewGitAnalyzer(repoPath string) *GitAnalyzer {
	return &GitAnalyzer{
		repoPath: repoPath,
	}
}

func (g *GitAnalyzer) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func (g *GitAnalyzer) GetFileMetrics(sinceDays int) (map[string]*model.GitMetrics, error) {
	metrics := make(map[string]*model.GitMetrics)

	args := []string{"log", "--pretty=format:%H|%an|%ad", "--name-only"}
	if sinceDays > 0 {
		since := time.Now().AddDate(0, 0, -sinceDays).Format("2006-01-02")
		args = append(args, "--since="+since)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var currentCommit struct {
		Hash   string
		Author string
		Date   string
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "|") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				currentCommit.Hash = parts[0]
				currentCommit.Author = parts[1]
				currentCommit.Date = parts[2]
			}
		} else {
			filePath := filepath.Join(g.repoPath, line)
			if _, ok := metrics[filePath]; !ok {
				metrics[filePath] = &model.GitMetrics{
					FilePath:    filePath,
					Authors:     []string{},
					LastCommit:  currentCommit.Date,
				}
			}
			m := metrics[filePath]
			m.CommitCount++
			m.LastCommit = currentCommit.Date

			authorExists := false
			for _, a := range m.Authors {
				if a == currentCommit.Author {
					authorExists = true
					break
				}
			}
			if !authorExists {
				m.Authors = append(m.Authors, currentCommit.Author)
			}
			m.AuthorCount = len(m.Authors)
		}
	}

	return metrics, nil
}

func (g *GitAnalyzer) GetChangedFiles(diffRange string) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if diffRange != "" {
		args = append(args, diffRange)
	} else {
		args = append(args, "HEAD")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.Join(g.repoPath, line))
		}
	}

	return files, nil
}

func (g *GitAnalyzer) GetBlame(filePath string) (map[int]string, error) {
	relPath, err := filepath.Rel(g.repoPath, filePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "blame", "--line-porcelain", relPath)
	cmd.Dir = g.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lineCommits := make(map[int]string)
	lines := strings.Split(string(output), "\n")
	currentLine := 0

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			if len(parts[0]) == 40 {
				if n, _ := fmt.Sscanf(parts[2], "%d", &currentLine); n == 1 {
					lineCommits[currentLine] = parts[0]
				}
			}
		}
	}

	return lineCommits, nil
}

func (g *GitAnalyzer) CalculateFunctionChangeFrequency(filePath string, functions []model.Function) map[string]int {
	result := make(map[string]int)

	lineCommits, err := g.GetBlame(filePath)
	if err != nil {
		return result
	}

	for _, fn := range functions {
		commitSet := make(map[string]bool)
		for line := fn.StartLine; line <= fn.EndLine; line++ {
			if commit, ok := lineCommits[line]; ok {
				commitSet[commit] = true
			}
		}
		result[fn.Name] = len(commitSet)
	}

	return result
}

func IsHotspot(gitMetrics *model.GitMetrics, avgCC int) bool {
	if gitMetrics == nil {
		return false
	}
	return gitMetrics.CommitCount >= 10 && avgCC >= 10
}

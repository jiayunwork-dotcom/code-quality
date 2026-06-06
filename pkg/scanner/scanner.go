package scanner

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/code-quality/cli/pkg/analysis"
	"github.com/code-quality/cli/pkg/baseline"
	"github.com/code-quality/cli/pkg/config"
	"github.com/code-quality/cli/pkg/git"
	"github.com/code-quality/cli/pkg/metrics"
	"github.com/code-quality/cli/pkg/model"
	"github.com/google/uuid"
)

type Scanner struct {
	projectRoot  string
	config       *model.Config
	ignoreRules  []model.QualityIgnoreRule
	gitAnalyzer  *git.GitAnalyzer
	baselineMgr  *baseline.Manager
	options      ScanOptions
}

type ScanOptions struct {
	SinceDays   int
	DiffRange   string
	FilesOnly   []string
	WithBaseline bool
	SaveBaseline bool
	FailOn      string
}

func NewScanner(projectRoot string, options ScanOptions) (*Scanner, error) {
	cfg, err := config.LoadConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	rules, err := config.LoadIgnoreRules(projectRoot)
	if err != nil {
		return nil, err
	}

	s := &Scanner{
		projectRoot: projectRoot,
		config:      cfg,
		ignoreRules: rules,
		gitAnalyzer: git.NewGitAnalyzer(projectRoot),
		baselineMgr: baseline.NewManager(projectRoot),
		options:     options,
	}

	return s, nil
}

func (s *Scanner) Scan() (*model.ProjectReport, error) {
	files, err := s.getFilesToScan()
	if err != nil {
		return nil, err
	}

	calc := metrics.NewCalculator(s.config)
	analyzer := analysis.NewArchitectureAnalyzer(s.config)

	var fileReports []*model.FileReport
	var fileReportPointers []*model.FileReport

	var gitMetrics map[string]*model.GitMetrics
	if s.gitAnalyzer.IsGitRepo() && s.config.Rules["change-hotspots"] {
		gitMetrics, _ = s.gitAnalyzer.GetFileMetrics(s.options.SinceDays)
	}

	for _, filePath := range files {
		if config.ShouldIgnore(filePath, s.ignoreRules, "") {
			continue
		}

		report, err := calc.CalculateFile(filePath)
		if err != nil {
			log.Printf("Warning: 解析文件失败 %s: %v\n", filePath, err)
			continue
		}
		if report == nil {
			continue
		}

		report.Violations = s.filterIgnoredViolations(filePath, report.Violations)

		if gm, ok := gitMetrics[filePath]; ok {
			report.GitMetrics = gm
			avgCC := s.averageCyclomaticComplexity(report)
			report.IsHotspot = git.IsHotspot(gm, avgCC)
		}

		fileReports = append(fileReports, report)
		fileReportPointers = append(fileReportPointers, report)
	}

	archIssues := analyzer.Analyze(fileReportPointers)

	report := &model.ProjectReport{
		Files:              fileReports,
		ArchitectureIssues: archIssues,
		GeneratedAt:        time.Now().Format(time.RFC3339),
	}

	report.Summary = s.calculateSummary(report)

	if s.options.WithBaseline && s.baselineMgr.Exists() {
		diff, err := s.baselineMgr.Compare(report)
		if err == nil {
			report.BaselineDiff = diff
		}
	}

	if s.options.SaveBaseline {
		if err := s.baselineMgr.Save(report); err != nil {
			log.Printf("Warning: 保存基线失败: %v\n", err)
		}
	}

	s.assignViolationIDs(report)

	return report, nil
}

func (s *Scanner) getFilesToScan() ([]string, error) {
	var files []string

	if len(s.options.FilesOnly) > 0 {
		for _, f := range s.options.FilesOnly {
			absPath, err := filepath.Abs(f)
			if err != nil {
				continue
			}
			if _, err := os.Stat(absPath); err == nil {
				files = append(files, absPath)
			}
		}
		return files, nil
	}

	if s.options.DiffRange != "" && s.gitAnalyzer.IsGitRepo() {
		changedFiles, err := s.gitAnalyzer.GetChangedFiles(s.options.DiffRange)
		if err == nil {
			for _, f := range changedFiles {
				if _, err := os.Stat(f); err == nil {
					files = append(files, f)
				}
			}
			return files, nil
		}
	}

	err := filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if config.IsIgnoredDir(path, s.config.IgnoreDirs) {
				return filepath.SkipDir
			}
			return nil
		}

		if isSupportedFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supported := map[string]bool{
		".go":   true,
		".py":   true,
		".js":   true,
		".jsx":  true,
		".ts":   true,
		".tsx":  true,
		".java": true,
	}
	return supported[ext]
}

func (s *Scanner) filterIgnoredViolations(filePath string, violations []model.Violation) []model.Violation {
	var filtered []model.Violation
	for _, v := range violations {
		if !config.ShouldIgnore(filePath, s.ignoreRules, v.RuleID) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func (s *Scanner) calculateSummary(report *model.ProjectReport) model.Summary {
	summary := model.Summary{
		TotalFiles: len(report.Files),
	}

	for _, fr := range report.Files {
		summary.TotalFunctions += len(fr.Functions)
		summary.TotalClasses += len(fr.Classes)
		summary.TotalViolations += len(fr.Violations)

		if fr.IsHotspot {
			summary.HotspotsCount++
		}

		for _, v := range fr.Violations {
			switch v.Severity {
			case model.SeverityCritical:
				summary.CriticalCount++
			case model.SeverityHigh:
				summary.HighCount++
			case model.SeverityMedium:
				summary.MediumCount++
			case model.SeverityLow:
				summary.LowCount++
			case model.SeverityInfo:
				summary.InfoCount++
			}
		}
	}

	return summary
}

func (s *Scanner) averageCyclomaticComplexity(report *model.FileReport) int {
	if len(report.Functions) == 0 {
		return 0
	}
	total := 0
	for _, f := range report.Functions {
		total += f.CyclomaticComplexity
	}
	return total / len(report.Functions)
}

func (s *Scanner) assignViolationIDs(report *model.ProjectReport) {
	for _, fr := range report.Files {
		for i := range fr.Violations {
			fr.Violations[i].ID = uuid.New().String()
		}
	}
	for i := range report.ArchitectureIssues {
		report.ArchitectureIssues[i].ID = uuid.New().String()
	}
}

func (s *Scanner) GetExitCode(report *model.ProjectReport) int {
	failOn := s.options.FailOn
	if failOn == "" {
		failOn = "critical,high"
	}

	failLevels := make(map[string]bool)
	for _, level := range strings.Split(failOn, ",") {
		failLevels[strings.TrimSpace(strings.ToLower(level))] = true
	}

	if failLevels["all"] || failLevels["warning"] {
		if report.Summary.TotalViolations > 0 {
			return 1
		}
	}

	if failLevels["critical"] && report.Summary.CriticalCount > 0 {
		return 1
	}

	if failLevels["high"] && report.Summary.HighCount > 0 {
		return 1
	}

	if failLevels["medium"] && report.Summary.MediumCount > 0 {
		return 1
	}

	return 0
}

package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/code-quality/cli/pkg/model"
)

const (
	BaselineDir  = ".code-quality"
	BaselineFile = "baseline.json"
)

type Manager struct {
	projectRoot string
}

func NewManager(projectRoot string) *Manager {
	return &Manager{
		projectRoot: projectRoot,
	}
}

func (m *Manager) BaselinePath() string {
	return filepath.Join(m.projectRoot, BaselineDir, BaselineFile)
}

func (m *Manager) Exists() bool {
	_, err := os.Stat(m.BaselinePath())
	return err == nil
}

func (m *Manager) Save(report *model.ProjectReport) error {
	dir := filepath.Join(m.projectRoot, BaselineDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	snapshot := &model.BaselineSnapshot{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Files:       make(map[string]model.FileBaseline),
	}

	for _, fr := range report.Files {
		if fr.File == nil {
			continue
		}
		fb := model.FileBaseline{
			Functions: make(map[string]model.FunctionMetrics),
		}
		for _, fm := range fr.Functions {
			fb.Functions[fm.FunctionName] = fm
		}
		snapshot.Files[fr.File.Path] = fb
		snapshot.Violations = append(snapshot.Violations, fr.Violations...)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.BaselinePath(), data, 0644)
}

func (m *Manager) Load() (*model.BaselineSnapshot, error) {
	data, err := os.ReadFile(m.BaselinePath())
	if err != nil {
		return nil, err
	}

	var snapshot model.BaselineSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (m *Manager) Compare(currentReport *model.ProjectReport) (*model.BaselineDiff, error) {
	baseline, err := m.Load()
	if err != nil {
		return nil, err
	}

	diff := &model.BaselineDiff{}

	baselineViolationMap := make(map[string]bool)
	for _, v := range baseline.Violations {
		key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
		baselineViolationMap[key] = true
	}

	for _, fr := range currentReport.Files {
		for _, v := range fr.Violations {
			key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
			if !baselineViolationMap[key] {
				diff.NewViolations = append(diff.NewViolations, v)
			}
		}
	}

	currentViolationMap := make(map[string]bool)
	for _, fr := range currentReport.Files {
		for _, v := range fr.Violations {
			key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
			currentViolationMap[key] = true
		}
	}

	for _, v := range baseline.Violations {
		key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
		if !currentViolationMap[key] {
			diff.FixedViolations = append(diff.FixedViolations, v)
		}
	}

	for _, fr := range currentReport.Files {
		if fr.File == nil {
			continue
		}
		if bf, ok := baseline.Files[fr.File.Path]; ok {
			for _, fm := range fr.Functions {
				if bfm, ok := bf.Functions[fm.FunctionName]; ok {
					if fm.CyclomaticComplexity > bfm.CyclomaticComplexity {
						diff.DeterioratedFuncs = append(diff.DeterioratedFuncs,
							fm.FunctionName+"@"+fr.File.Path)
					} else if fm.CyclomaticComplexity < bfm.CyclomaticComplexity {
						diff.ImprovedFuncs = append(diff.ImprovedFuncs,
							fm.FunctionName+"@"+fr.File.Path)
					}
				}
			}
		}
	}

	return diff, nil
}

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

func (m *Manager) CompareIncremental(currentReport *model.ProjectReport, changedFiles []string) (*model.IncrementalDiff, error) {
	baseline, err := m.Load()
	if err != nil {
		return nil, err
	}

	diff := &model.IncrementalDiff{
		FunctionChanges: []model.FunctionChange{},
		NewViolations:   []model.Violation{},
	}

	baselineViolationMap := make(map[string]bool)
	for _, v := range baseline.Violations {
		key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
		baselineViolationMap[key] = true
	}

	changedFileSet := make(map[string]bool)
	for _, f := range changedFiles {
		changedFileSet[f] = true
	}

	for _, fr := range currentReport.Files {
		if fr.File == nil {
			continue
		}
		if !changedFileSet[fr.File.Path] {
			continue
		}
		for _, v := range fr.Violations {
			key := v.RuleID + ":" + v.FilePath + ":" + string(rune(v.StartLine))
			if !baselineViolationMap[key] {
				diff.NewViolations = append(diff.NewViolations, v)
			}
		}
	}

	for _, fr := range currentReport.Files {
		if fr.File == nil {
			continue
		}
		if !changedFileSet[fr.File.Path] {
			continue
		}
		filePath := fr.File.Path
		bf, hasBaselineFile := baseline.Files[filePath]

		currentFuncMap := make(map[string]model.FunctionMetrics)
		for _, fm := range fr.Functions {
			currentFuncMap[fm.FunctionName] = fm
		}

		baselineFuncMap := make(map[string]model.FunctionMetrics)
		if hasBaselineFile {
			for name, fm := range bf.Functions {
				baselineFuncMap[name] = fm
			}
		}

		for name, fm := range currentFuncMap {
			if _, exists := baselineFuncMap[name]; !exists {
				diff.FunctionChanges = append(diff.FunctionChanges, model.FunctionChange{
					FilePath:     filePath,
					FunctionName: name,
					ChangeType:   model.FuncChangeAdded,
				})
				continue
			}
		}

		if hasBaselineFile {
			for name := range baselineFuncMap {
				if _, exists := currentFuncMap[name]; !exists {
					diff.FunctionChanges = append(diff.FunctionChanges, model.FunctionChange{
						FilePath:     filePath,
						FunctionName: name,
						ChangeType:   model.FuncChangeRemoved,
					})
				}
			}
		}

		if hasBaselineFile {
			for name, currentFM := range currentFuncMap {
				baselineFM, exists := baselineFuncMap[name]
				if !exists {
					continue
				}
				ccDelta := currentFM.CyclomaticComplexity - baselineFM.CyclomaticComplexity
				cogDelta := currentFM.CognitiveComplexity - baselineFM.CognitiveComplexity
				locDelta := currentFM.LOC - baselineFM.LOC

				hasChange := ccDelta != 0 || cogDelta != 0 || locDelta != 0
				if !hasChange {
					continue
				}

				isDeteriorated := ccDelta > 0 || cogDelta > 0

				changeType := model.FuncChangeImproved
				if isDeteriorated {
					changeType = model.FuncChangeDeteriorated
				}

				fc := model.FunctionChange{
					FilePath:     filePath,
					FunctionName: name,
					ChangeType:   changeType,
				}

				if ccDelta != 0 {
					fc.CyclomaticChange = &model.MetricChange{
						Before: baselineFM.CyclomaticComplexity,
						After:  currentFM.CyclomaticComplexity,
						Delta:  ccDelta,
					}
				}
				if cogDelta != 0 {
					fc.CognitiveChange = &model.MetricChange{
						Before: baselineFM.CognitiveComplexity,
						After:  currentFM.CognitiveComplexity,
						Delta:  cogDelta,
					}
				}
				if locDelta != 0 {
					fc.LOCChange = &model.MetricChange{
						Before: baselineFM.LOC,
						After:  currentFM.LOC,
						Delta:  locDelta,
					}
				}

				diff.FunctionChanges = append(diff.FunctionChanges, fc)
			}
		}
	}

	return diff, nil
}

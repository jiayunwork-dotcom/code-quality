package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/code-quality/cli/pkg/model"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = ".code-quality.yml"
	DefaultIgnoreFile = ".qualityignore"
)

func LoadConfig(projectRoot string) (*model.Config, error) {
	config := model.DefaultConfig()

	configPath := filepath.Join(projectRoot, DefaultConfigFile)
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		fileConfig := model.DefaultConfig()
		if err := yaml.Unmarshal(data, fileConfig); err != nil {
			return nil, err
		}

		mergeConfig(config, fileConfig)
	}

	return config, nil
}

func mergeConfig(base, override *model.Config) {
	if override.Thresholds.CyclomaticComplexityYellow > 0 {
		base.Thresholds.CyclomaticComplexityYellow = override.Thresholds.CyclomaticComplexityYellow
	}
	if override.Thresholds.CyclomaticComplexityRed > 0 {
		base.Thresholds.CyclomaticComplexityRed = override.Thresholds.CyclomaticComplexityRed
	}
	if override.Thresholds.CognitiveComplexityYellow > 0 {
		base.Thresholds.CognitiveComplexityYellow = override.Thresholds.CognitiveComplexityYellow
	}
	if override.Thresholds.CognitiveComplexityRed > 0 {
		base.Thresholds.CognitiveComplexityRed = override.Thresholds.CognitiveComplexityRed
	}
	if override.Thresholds.FunctionLOCYellow > 0 {
		base.Thresholds.FunctionLOCYellow = override.Thresholds.FunctionLOCYellow
	}
	if override.Thresholds.FunctionLOCRed > 0 {
		base.Thresholds.FunctionLOCRed = override.Thresholds.FunctionLOCRed
	}
	if override.Thresholds.ParamCountYellow > 0 {
		base.Thresholds.ParamCountYellow = override.Thresholds.ParamCountYellow
	}
	if override.Thresholds.ParamCountRed > 0 {
		base.Thresholds.ParamCountRed = override.Thresholds.ParamCountRed
	}
	if override.Thresholds.ClassMethodCountYellow > 0 {
		base.Thresholds.ClassMethodCountYellow = override.Thresholds.ClassMethodCountYellow
	}
	if override.Thresholds.ClassMethodCountRed > 0 {
		base.Thresholds.ClassMethodCountRed = override.Thresholds.ClassMethodCountRed
	}
	if override.Thresholds.CBOYellow > 0 {
		base.Thresholds.CBOYellow = override.Thresholds.CBOYellow
	}
	if override.Thresholds.CBORed > 0 {
		base.Thresholds.CBORed = override.Thresholds.CBORed
	}
	if override.Thresholds.RFCYellow > 0 {
		base.Thresholds.RFCYellow = override.Thresholds.RFCYellow
	}
	if override.Thresholds.RFCRed > 0 {
		base.Thresholds.RFCRed = override.Thresholds.RFCRed
	}

	if len(override.IgnoreDirs) > 0 {
		base.IgnoreDirs = append(base.IgnoreDirs, override.IgnoreDirs...)
	}

	if len(override.Layers) > 0 {
		base.Layers = override.Layers
	}

	if override.Rules != nil {
		for k, v := range override.Rules {
			base.Rules[k] = v
		}
	}
}

func LoadIgnoreRules(projectRoot string) ([]model.QualityIgnoreRule, error) {
	var rules []model.QualityIgnoreRule

	ignorePath := filepath.Join(projectRoot, DefaultIgnoreFile)
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		return rules, nil
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := model.QualityIgnoreRule{}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == "ignore" {
			ruleParts := strings.SplitN(parts[1], " ", 2)
			if len(ruleParts) == 2 {
				rule.Pattern = strings.TrimSpace(ruleParts[1])
				rule.RuleIDs = strings.Split(ruleParts[0], ",")
			} else {
				rule.Pattern = strings.TrimSpace(parts[1])
			}
		} else {
			if strings.HasPrefix(line, "!") {
				rule.Negate = true
				rule.Pattern = strings.TrimPrefix(line, "!")
			} else {
				rule.Pattern = line
			}
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func ShouldIgnore(filePath string, rules []model.QualityIgnoreRule, ruleID string) bool {
	ignored := false

	for _, rule := range rules {
		matched, _ := filepath.Match(rule.Pattern, filepath.Base(filePath))
		if !matched {
			matched, _ = filepath.Match(rule.Pattern, filePath)
		}
		if !matched {
			matched = strings.Contains(filePath, rule.Pattern)
		}

		if matched {
			if rule.Negate {
				ignored = false
			} else {
				if len(rule.RuleIDs) == 0 {
					ignored = true
				} else {
					for _, rid := range rule.RuleIDs {
						if rid == ruleID {
							ignored = true
							break
						}
					}
				}
			}
		}
	}

	return ignored
}

func IsIgnoredDir(path string, ignoreDirs []string) bool {
	base := filepath.Base(path)
	for _, dir := range ignoreDirs {
		if base == dir {
			return true
		}
		if strings.Contains(path, string(filepath.Separator)+dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

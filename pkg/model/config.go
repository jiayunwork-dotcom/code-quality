package model

type Thresholds struct {
	CyclomaticComplexityYellow int `yaml:"cyclomatic_complexity_yellow" json:"cyclomatic_complexity_yellow"`
	CyclomaticComplexityRed    int `yaml:"cyclomatic_complexity_red" json:"cyclomatic_complexity_red"`
	CognitiveComplexityYellow  int `yaml:"cognitive_complexity_yellow" json:"cognitive_complexity_yellow"`
	CognitiveComplexityRed     int `yaml:"cognitive_complexity_red" json:"cognitive_complexity_red"`
	FunctionLOCYellow          int `yaml:"function_loc_yellow" json:"function_loc_yellow"`
	FunctionLOCRed             int `yaml:"function_loc_red" json:"function_loc_red"`
	ParamCountYellow           int `yaml:"param_count_yellow" json:"param_count_yellow"`
	ParamCountRed              int `yaml:"param_count_red" json:"param_count_red"`
	ClassMethodCountYellow     int `yaml:"class_method_count_yellow" json:"class_method_count_yellow"`
	ClassMethodCountRed        int `yaml:"class_method_count_red" json:"class_method_count_red"`
	CBOYellow                  int `yaml:"cbo_yellow" json:"cbo_yellow"`
	CBORed                     int `yaml:"cbo_red" json:"cbo_red"`
	RFCYellow                  int `yaml:"rfc_yellow" json:"rfc_yellow"`
	RFCRed                     int `yaml:"rfc_red" json:"rfc_red"`
}

type LayerRule struct {
	Name       string   `yaml:"name" json:"name"`
	Paths      []string `yaml:"paths" json:"paths"`
	MayDepend  []string `yaml:"may_depend" json:"may_depend"`
}

type Config struct {
	Thresholds     Thresholds            `yaml:"thresholds" json:"thresholds"`
	IgnoreDirs     []string              `yaml:"ignore_dirs" json:"ignore_dirs"`
	IgnorePatterns []string              `yaml:"ignore_patterns" json:"ignore_patterns"`
	Layers         []LayerRule           `yaml:"layers" json:"layers"`
	Rules          map[string]bool       `yaml:"rules" json:"rules"`
	PathOverrides  map[string]*Config    `yaml:"path_overrides" json:"path_overrides"`
}

func DefaultConfig() *Config {
	return &Config{
		Thresholds: Thresholds{
			CyclomaticComplexityYellow: 10,
			CyclomaticComplexityRed:    20,
			CognitiveComplexityYellow:  15,
			CognitiveComplexityRed:     25,
			FunctionLOCYellow:          50,
			FunctionLOCRed:             100,
			ParamCountYellow:           5,
			ParamCountRed:              8,
			ClassMethodCountYellow:     15,
			ClassMethodCountRed:        20,
			CBOYellow:                  10,
			CBORed:                     20,
			RFCYellow:                  30,
			RFCRed:                     50,
		},
		IgnoreDirs: []string{
			"vendor",
			"node_modules",
			"generated",
			"dist",
			"build",
			".git",
			"__pycache__",
			".idea",
			".vscode",
		},
		Rules: map[string]bool{
			"cyclomatic-complexity":  true,
			"cognitive-complexity":   true,
			"function-length":        true,
			"long-parameter-list":    true,
			"god-class":              true,
			"feature-envy":           true,
			"data-clumps":            true,
			"cyclic-dependencies":    true,
			"layer-violations":       true,
			"change-hotspots":        true,
		},
	}
}

type QualityIgnoreRule struct {
	Pattern     string
	RuleIDs     []string
	Negate      bool
}

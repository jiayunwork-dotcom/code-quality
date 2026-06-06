package model

type Severity string

const (
	SeverityNone   Severity = "none"
	SeverityInfo   Severity = "info"
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
	SeverityCritical Severity = "critical"
)

type FunctionMetrics struct {
	FilePath           string  `json:"file_path"`
	FunctionName       string  `json:"function_name"`
	StartLine          int     `json:"start_line"`
	EndLine            int     `json:"end_line"`
	CyclomaticComplexity int   `json:"cyclomatic_complexity"`
	CognitiveComplexity  int   `json:"cognitive_complexity"`
	LOC                int     `json:"loc"`
	ParamCount         int     `json:"param_count"`
	NestingDepth       int     `json:"nesting_depth"`
}

type ClassMetrics struct {
	FilePath    string `json:"file_path"`
	ClassName   string `json:"class_name"`
	MethodCount int    `json:"method_count"`
	AttrCount   int    `json:"attr_count"`
	CBO         int    `json:"cbo"`
	RFC         int    `json:"rfc"`
	LOC         int    `json:"loc"`
}

type Violation struct {
	ID         string   `json:"id"`
	RuleID     string   `json:"rule_id"`
	Severity   Severity `json:"severity"`
	Message    string   `json:"message"`
	FilePath   string   `json:"file_path"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type ArchitectureIssue struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Severity   Severity `json:"severity"`
	Message    string   `json:"message"`
	Details    []string `json:"details,omitempty"`
	FilePath   string   `json:"file_path,omitempty"`
}

type GitMetrics struct {
	FilePath     string   `json:"file_path"`
	CommitCount  int      `json:"commit_count"`
	AuthorCount  int      `json:"author_count"`
	LastCommit   string   `json:"last_commit"`
	Authors      []string `json:"authors,omitempty"`
	FunctionChanges map[string]int `json:"function_changes,omitempty"`
}

type FileReport struct {
	File        *File            `json:"file"`
	Functions   []FunctionMetrics `json:"functions"`
	Classes     []ClassMetrics   `json:"classes"`
	Violations  []Violation      `json:"violations"`
	GitMetrics  *GitMetrics      `json:"git_metrics,omitempty"`
	IsHotspot   bool             `json:"is_hotspot"`
}

type ProjectReport struct {
	Files              []FileReport         `json:"files"`
	ArchitectureIssues []ArchitectureIssue  `json:"architecture_issues"`
	Summary            Summary              `json:"summary"`
	GeneratedAt        string               `json:"generated_at"`
	BaselineDiff       *BaselineDiff        `json:"baseline_diff,omitempty"`
}

type Summary struct {
	TotalFiles       int `json:"total_files"`
	TotalFunctions   int `json:"total_functions"`
	TotalClasses     int `json:"total_classes"`
	TotalViolations  int `json:"total_violations"`
	CriticalCount    int `json:"critical_count"`
	HighCount        int `json:"high_count"`
	MediumCount      int `json:"medium_count"`
	LowCount         int `json:"low_count"`
	InfoCount        int `json:"info_count"`
	HotspotsCount    int `json:"hotspots_count"`
}

type BaselineDiff struct {
	NewViolations     []Violation `json:"new_violations"`
	FixedViolations   []Violation `json:"fixed_violations"`
	DeterioratedFuncs []string    `json:"deteriorated_functions"`
	ImprovedFuncs     []string    `json:"improved_functions"`
}

type BaselineSnapshot struct {
	GeneratedAt string                  `json:"generated_at"`
	Files       map[string]FileBaseline `json:"files"`
	Violations  []Violation             `json:"violations"`
}

type FileBaseline struct {
	Functions map[string]FunctionMetrics `json:"functions"`
}

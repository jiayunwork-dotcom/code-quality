package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/code-quality/cli/pkg/model"
	"github.com/fatih/color"
)

type Format string

const (
	FormatText   Format = "text"
	FormatJSON   Format = "json"
	FormatHTML   Format = "html"
	FormatSARIF  Format = "sarif"
)

type Reporter struct {
	format Format
	output string
}

func NewReporter(format Format, output string) *Reporter {
	return &Reporter{
		format: format,
		output: output,
	}
}

func (r *Reporter) Generate(report *model.ProjectReport) error {
	if report.BaselineDiff != nil && report.BaselineDiff.IncrementalDiff != nil {
		switch r.format {
		case FormatText:
			return r.generateIncrementalText(report)
		case FormatJSON:
			return r.generateJSON(report)
		case FormatHTML:
			return r.generateHTML(report)
		case FormatSARIF:
			return r.generateSARIF(report)
		default:
			return r.generateIncrementalText(report)
		}
	}
	switch r.format {
	case FormatText:
		return r.generateText(report)
	case FormatJSON:
		return r.generateJSON(report)
	case FormatHTML:
		return r.generateHTML(report)
	case FormatSARIF:
		return r.generateSARIF(report)
	default:
		return r.generateText(report)
	}
}

func (r *Reporter) generateText(report *model.ProjectReport) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	color.Cyan("\n=== 代码质量分析报告 ===\n")

	fmt.Fprintf(w, "总文件数:\t%d\n", report.Summary.TotalFiles)
	fmt.Fprintf(w, "总函数数:\t%d\n", report.Summary.TotalFunctions)
	fmt.Fprintf(w, "总类数:\t%d\n", report.Summary.TotalClasses)
	fmt.Fprintf(w, "违规总数:\t%d\n", report.Summary.TotalViolations)
	w.Flush()

	color.Yellow("\n--- 违规统计 ---\n")
	if report.Summary.CriticalCount > 0 {
		color.Red("严重(Critical): %d\n", report.Summary.CriticalCount)
	}
	if report.Summary.HighCount > 0 {
		color.Magenta("高(High): %d\n", report.Summary.HighCount)
	}
	if report.Summary.MediumCount > 0 {
		color.Yellow("中(Medium): %d\n", report.Summary.MediumCount)
	}
	if report.Summary.LowCount > 0 {
		color.Green("低(Low): %d\n", report.Summary.LowCount)
	}

	if report.Summary.HotspotsCount > 0 {
		color.Red("\n⚠ 变更热点: %d 个文件\n", report.Summary.HotspotsCount)
	}

	if len(report.Files) > 0 {
		color.Cyan("\n--- 函数复杂度 Top 10 ---\n")
		var allFuncs []model.FunctionMetrics
		for _, fr := range report.Files {
			allFuncs = append(allFuncs, fr.Functions...)
		}
		sort.Slice(allFuncs, func(i, j int) bool {
			return allFuncs[i].CyclomaticComplexity > allFuncs[j].CyclomaticComplexity
		})
		if len(allFuncs) > 10 {
			allFuncs = allFuncs[:10]
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "函数\t文件\t圈复杂度\t认知复杂度\t行数\t参数")
		fmt.Fprintln(tw, "----\t----\t----------\t----------\t----\t----")
		for _, f := range allFuncs {
			ccColor := color.GreenString
			if f.CyclomaticComplexity >= 20 {
				ccColor = color.RedString
			} else if f.CyclomaticComplexity >= 10 {
				ccColor = color.YellowString
			}

			relPath, _ := filepath.Rel(".", f.FilePath)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%d\n",
				f.FunctionName,
				relPath,
				ccColor("%d", f.CyclomaticComplexity),
				f.CognitiveComplexity,
				f.LOC,
				f.ParamCount,
			)
		}
		tw.Flush()
	}

	if len(report.ArchitectureIssues) > 0 {
		color.Red("\n--- 架构问题 ---\n")
		for _, issue := range report.ArchitectureIssues {
			switch issue.Severity {
			case model.SeverityCritical, model.SeverityHigh:
				color.Red("  [%s] %s\n", issue.Type, issue.Message)
			case model.SeverityMedium:
				color.Yellow("  [%s] %s\n", issue.Type, issue.Message)
			default:
				color.White("  [%s] %s\n", issue.Type, issue.Message)
			}
			for _, d := range issue.Details {
				fmt.Printf("    - %s\n", d)
			}
		}
	}

	if len(report.Files) > 0 {
		color.Magenta("\n--- 违规详情 ---\n")
		for _, fr := range report.Files {
			if len(fr.Violations) == 0 {
				continue
			}
			relPath, _ := filepath.Rel(".", fr.File.Path)
			fmt.Printf("\n文件: %s\n", relPath)
			for _, v := range fr.Violations {
				sevColor := color.WhiteString
				switch v.Severity {
				case model.SeverityCritical:
					sevColor = color.RedString
				case model.SeverityHigh:
					sevColor = color.MagentaString
				case model.SeverityMedium:
					sevColor = color.YellowString
				case model.SeverityLow:
					sevColor = color.GreenString
				}
				fmt.Printf("  %s L%d-%d: [%s] %s\n",
					sevColor(strings.ToUpper(string(v.Severity))),
					v.StartLine, v.EndLine, v.RuleID, v.Message)
				if v.Suggestion != "" {
					color.Green("    建议: %s\n", v.Suggestion)
				}
			}
		}
	}

	if report.BaselineDiff != nil {
		color.Cyan("\n--- 基线对比 ---\n")
		if len(report.BaselineDiff.NewViolations) > 0 {
			color.Red("新增违规: %d 项\n", len(report.BaselineDiff.NewViolations))
		}
		if len(report.BaselineDiff.FixedViolations) > 0 {
			color.Green("已修复: %d 项\n", len(report.BaselineDiff.FixedViolations))
		}
		if len(report.BaselineDiff.DeterioratedFuncs) > 0 {
			color.Yellow("恶化函数: %d 个\n", len(report.BaselineDiff.DeterioratedFuncs))
		}
		if len(report.BaselineDiff.ImprovedFuncs) > 0 {
			color.Green("改善函数: %d 个\n", len(report.BaselineDiff.ImprovedFuncs))
		}
	}

	return nil
}

func (r *Reporter) generateJSON(report *model.ProjectReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if r.output != "" && r.output != "-" {
		return os.WriteFile(r.output, data, 0644)
	}
	fmt.Println(string(data))
	return nil
}

func (r *Reporter) generateHTML(report *model.ProjectReport) error {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>代码质量分析报告</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 20px; }
        .summary { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; }
        .stat { text-align: center; padding: 15px; background: #f8f9fa; border-radius: 6px; }
        .stat-value { font-size: 24px; font-weight: bold; color: #333; }
        .stat-label { font-size: 12px; color: #666; margin-top: 5px; }
        .critical { color: #dc3545 !important; }
        .high { color: #fd7e14 !important; }
        .medium { color: #ffc107 !important; }
        .low { color: #28a745 !important; }
        .section { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .section h2 { color: #333; margin-bottom: 15px; font-size: 18px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #f8f9fa; font-weight: 600; }
        tr:hover { background: #f8f9fa; }
        .badge { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 12px; font-weight: 500; }
        .badge-critical { background: #dc3545; color: white; }
        .badge-high { background: #fd7e14; color: white; }
        .badge-medium { background: #ffc107; color: #333; }
        .badge-low { background: #28a745; color: white; }
        .file-tree { max-height: 400px; overflow-y: auto; }
        .file-item { padding: 8px 12px; cursor: pointer; border-radius: 4px; }
        .file-item:hover { background: #f0f0f0; }
        .hotspot { background: #fff3cd !important; }
    </style>
</head>
<body>
    <div class="container">
        <h1>代码质量分析报告</h1>
        
        <div class="summary">
            <div class="summary-grid">
                <div class="stat">
                    <div class="stat-value">` + fmt.Sprint(report.Summary.TotalFiles) + `</div>
                    <div class="stat-label">总文件数</div>
                </div>
                <div class="stat">
                    <div class="stat-value">` + fmt.Sprint(report.Summary.TotalFunctions) + `</div>
                    <div class="stat-label">总函数数</div>
                </div>
                <div class="stat">
                    <div class="stat-value">` + fmt.Sprint(report.Summary.TotalClasses) + `</div>
                    <div class="stat-label">总类数</div>
                </div>
                <div class="stat">
                    <div class="stat-value critical">` + fmt.Sprint(report.Summary.CriticalCount) + `</div>
                    <div class="stat-label">严重违规</div>
                </div>
                <div class="stat">
                    <div class="stat-value high">` + fmt.Sprint(report.Summary.HighCount) + `</div>
                    <div class="stat-label">高优先级</div>
                </div>
                <div class="stat">
                    <div class="stat-value medium">` + fmt.Sprint(report.Summary.MediumCount) + `</div>
                    <div class="stat-label">中优先级</div>
                </div>
            </div>
        </div>
`

	if len(report.ArchitectureIssues) > 0 {
		html += `
        <div class="section">
            <h2>架构问题</h2>
            <table>
                <tr><th>类型</th><th>严重程度</th><th>描述</th><th>详情</th></tr>
`
		for _, issue := range report.ArchitectureIssues {
			badgeClass := "badge-low"
			switch issue.Severity {
			case model.SeverityCritical:
				badgeClass = "badge-critical"
			case model.SeverityHigh:
				badgeClass = "badge-high"
			case model.SeverityMedium:
				badgeClass = "badge-medium"
			}
			details := strings.Join(issue.Details, ", ")
			html += fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td><span class="badge %s">%s</span></td>
                    <td>%s</td>
                    <td>%s</td>
                </tr>`, issue.Type, badgeClass, issue.Severity, issue.Message, details)
		}
		html += `
            </table>
        </div>
`
	}

	html += `
        <div class="section">
            <h2>函数复杂度排行</h2>
            <table>
                <tr><th>函数名</th><th>文件</th><th>圈复杂度</th><th>认知复杂度</th><th>行数</th><th>参数</th></tr>
`
	var allFuncs []model.FunctionMetrics
	for _, fr := range report.Files {
		allFuncs = append(allFuncs, fr.Functions...)
	}
	sort.Slice(allFuncs, func(i, j int) bool {
		return allFuncs[i].CyclomaticComplexity > allFuncs[j].CyclomaticComplexity
	})
	for _, f := range allFuncs {
		ccClass := "low"
		if f.CyclomaticComplexity >= 20 {
			ccClass = "critical"
		} else if f.CyclomaticComplexity >= 10 {
			ccClass = "medium"
		}
		relPath, _ := filepath.Rel(".", f.FilePath)
		html += fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td>%s</td>
                    <td class="%s">%d</td>
                    <td>%d</td>
                    <td>%d</td>
                    <td>%d</td>
                </tr>`, f.FunctionName, relPath, ccClass, f.CyclomaticComplexity, f.CognitiveComplexity, f.LOC, f.ParamCount)
	}

	html += `
            </table>
        </div>
    </div>
</body>
</html>
`

	if r.output != "" && r.output != "-" {
		return os.WriteFile(r.output, []byte(html), 0644)
	}
	fmt.Println(html)
	return nil
}

type SARIFReport struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

type Tool struct {
	Driver Driver `json:"driver"`
}

type Driver struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	InformationURI string `json:"informationUri"`
	Rules          []Rule `json:"rules"`
}

type Rule struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ShortDescription Description `json:"shortDescription"`
	FullDescription  Description `json:"fullDescription"`
	DefaultConfiguration Configuration `json:"defaultConfiguration"`
}

type Description struct {
	Text string `json:"text"`
}

type Configuration struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level"`
}

type Result struct {
	RuleID     string      `json:"ruleId"`
	Level      string      `json:"level"`
	Message    Description `json:"message"`
	Locations  []Location  `json:"locations"`
}

type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

type ArtifactLocation struct {
	URI string `json:"uri"`
}

type Region struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
}

func (r *Reporter) generateSARIF(report *model.ProjectReport) error {
	rulesMap := make(map[string]bool)
	var results []Result

	for _, fr := range report.Files {
		for _, v := range fr.Violations {
			rulesMap[v.RuleID] = true

			level := "note"
			switch v.Severity {
			case model.SeverityCritical:
				level = "error"
			case model.SeverityHigh:
				level = "error"
			case model.SeverityMedium:
				level = "warning"
			case model.SeverityLow:
				level = "note"
			}

			relPath, _ := filepath.Rel(".", v.FilePath)
			results = append(results, Result{
				RuleID:  v.RuleID,
				Level:   level,
				Message: Description{Text: v.Message},
				Locations: []Location{
					{
						PhysicalLocation: PhysicalLocation{
							ArtifactLocation: ArtifactLocation{URI: relPath},
							Region: Region{
								StartLine: v.StartLine,
								EndLine:   v.EndLine,
							},
						},
					},
				},
			})
		}
	}

	var rules []Rule
	for ruleID := range rulesMap {
		level := "warning"
		rules = append(rules, Rule{
			ID:   ruleID,
			Name: ruleID,
			ShortDescription: Description{Text: ruleID},
			FullDescription:  Description{Text: ruleID + "检测规则"},
			DefaultConfiguration: Configuration{
				Enabled: true,
				Level:   level,
			},
		})
	}

	sarif := SARIFReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []Run{
			{
				Tool: Tool{
					Driver: Driver{
						Name:           "code-quality-cli",
						Version:        "1.0.0",
						InformationURI: "https://github.com/code-quality/cli",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return err
	}

	if r.output != "" && r.output != "-" {
		return os.WriteFile(r.output, data, 0644)
	}
	fmt.Println(string(data))
	return nil
}

func (r *Reporter) generateIncrementalText(report *model.ProjectReport) error {
	color.Cyan("\n=== 增量代码质量分析报告 ===\n")

	if report.BaselineDiff == nil || report.BaselineDiff.IncrementalDiff == nil {
		fmt.Println("无增量对比数据")
		return nil
	}

	incDiff := report.BaselineDiff.IncrementalDiff

	fmt.Printf("分析文件数: %d\n", len(report.Files))
	fmt.Printf("函数变化数: %d\n", len(incDiff.FunctionChanges))
	fmt.Printf("新增违规数: %d\n", len(incDiff.NewViolations))

	if len(incDiff.FunctionChanges) > 0 {
		color.Yellow("\n--- 函数变化详情 ---\n")
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "文件名\t函数名\t变化类型\t指标变化详情")
		fmt.Fprintln(tw, "------\t------\t--------\t------------")

		for _, fc := range incDiff.FunctionChanges {
			relPath, _ := filepath.Rel(".", fc.FilePath)
			changeType := ""
			typeColor := color.WhiteString
			switch fc.ChangeType {
			case model.FuncChangeAdded:
				changeType = "新增"
				typeColor = color.YellowString
			case model.FuncChangeRemoved:
				changeType = "删除"
				typeColor = color.YellowString
			case model.FuncChangeDeteriorated:
				changeType = "恶化"
				typeColor = color.RedString
			case model.FuncChangeImproved:
				changeType = "改善"
				typeColor = color.GreenString
			}

			details := r.formatMetricChanges(fc)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				relPath,
				fc.FunctionName,
				typeColor(changeType),
				details,
			)
		}
		tw.Flush()
	}

	if len(incDiff.NewViolations) > 0 {
		color.Red("\n--- 新增违规详情 ---\n")
		for _, v := range incDiff.NewViolations {
			relPath, _ := filepath.Rel(".", v.FilePath)
			sevColor := color.WhiteString
			switch v.Severity {
			case model.SeverityCritical:
				sevColor = color.RedString
			case model.SeverityHigh:
				sevColor = color.MagentaString
			case model.SeverityMedium:
				sevColor = color.YellowString
			case model.SeverityLow:
				sevColor = color.GreenString
			}
			fmt.Printf("  %s L%d-%d: [%s] %s\n",
				relPath,
				v.StartLine, v.EndLine,
				sevColor(strings.ToUpper(string(v.Severity))),
				v.Message)
		}
	}

	addedCount := 0
	removedCount := 0
	deterioratedCount := 0
	improvedCount := 0
	for _, fc := range incDiff.FunctionChanges {
		switch fc.ChangeType {
		case model.FuncChangeAdded:
			addedCount++
		case model.FuncChangeRemoved:
			removedCount++
		case model.FuncChangeDeteriorated:
			deterioratedCount++
		case model.FuncChangeImproved:
			improvedCount++
		}
	}

	color.Cyan("\n--- 统计汇总 ---\n")
	fmt.Printf("  新增函数: %d 个\n", addedCount)
	fmt.Printf("  删除函数: %d 个\n", removedCount)
	if deterioratedCount > 0 {
		color.Red("  指标恶化: %d 个\n", deterioratedCount)
	} else {
		fmt.Printf("  指标恶化: %d 个\n", deterioratedCount)
	}
	if improvedCount > 0 {
		color.Green("  指标改善: %d 个\n", improvedCount)
	} else {
		fmt.Printf("  指标改善: %d 个\n", improvedCount)
	}

	return nil
}

func (r *Reporter) formatMetricChanges(fc model.FunctionChange) string {
	var parts []string

	isAdded := fc.ChangeType == model.FuncChangeAdded

	if fc.CyclomaticChange != nil {
		cc := fc.CyclomaticChange
		if isAdded && cc.Before == 0 {
			parts = append(parts, fmt.Sprintf("圈复杂度 %d", cc.After))
		} else {
			direction := "↑"
			if cc.Delta < 0 {
				direction = "↓"
			}
			parts = append(parts, fmt.Sprintf("圈复杂度 %d→%d (%s%d)",
				cc.Before, cc.After, direction, abs(cc.Delta)))
		}
	}
	if fc.CognitiveChange != nil {
		cog := fc.CognitiveChange
		if isAdded && cog.Before == 0 {
			parts = append(parts, fmt.Sprintf("认知复杂度 %d", cog.After))
		} else {
			direction := "↑"
			if cog.Delta < 0 {
				direction = "↓"
			}
			parts = append(parts, fmt.Sprintf("认知复杂度 %d→%d (%s%d)",
				cog.Before, cog.After, direction, abs(cog.Delta)))
		}
	}
	if fc.LOCChange != nil {
		loc := fc.LOCChange
		if isAdded && loc.Before == 0 {
			parts = append(parts, fmt.Sprintf("行数 %d", loc.After))
		} else {
			direction := "↑"
			if loc.Delta < 0 {
				direction = "↓"
			}
			parts = append(parts, fmt.Sprintf("行数 %d→%d (%s%d)",
				loc.Before, loc.After, direction, abs(loc.Delta)))
		}
	}

	if len(parts) == 0 {
		switch fc.ChangeType {
		case model.FuncChangeAdded:
			return "新增函数"
		case model.FuncChangeRemoved:
			return "删除函数"
		}
	}

	return strings.Join(parts, ", ")
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

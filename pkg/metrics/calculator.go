package metrics

import (
	"os"
	"strings"

	"github.com/code-quality/cli/pkg/model"
	"github.com/code-quality/cli/pkg/parser"
)

type Calculator struct {
	config *model.Config
}

func NewCalculator(config *model.Config) *Calculator {
	return &Calculator{
		config: config,
	}
}

func (c *Calculator) CalculateFile(filePath string) (*model.FileReport, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseFile(filePath, content)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}

	report := &model.FileReport{
		File: file,
	}

	lines := parser.SplitLines(string(content))
	report.Functions = c.calculateFunctionMetrics(file, lines)
	report.Classes = c.calculateClassMetrics(file, lines)
	report.Violations = c.checkViolations(file, report.Functions, report.Classes)

	return report, nil
}

func (c *Calculator) calculateFunctionMetrics(file *model.File, lines []string) []model.FunctionMetrics {
	var metrics []model.FunctionMetrics
	commentPrefix := getCommentPrefix(file.Language)

	for _, fn := range file.Functions {
		funcLines := lines[fn.StartLine-1 : fn.EndLine]
		loc := parser.CountEffectiveLOC(funcLines, commentPrefix...)

		cc := c.calculateCyclomaticComplexity(funcLines, file.Language)
		cog := c.calculateCognitiveComplexity(funcLines, file.Language)

		metrics = append(metrics, model.FunctionMetrics{
			FilePath:           file.Path,
			FunctionName:       fn.Name,
			StartLine:          fn.StartLine,
			EndLine:            fn.EndLine,
			CyclomaticComplexity: cc,
			CognitiveComplexity:  cog,
			LOC:                loc,
			ParamCount:         len(fn.Params),
			NestingDepth:       fn.NestingDepth,
		})
	}

	return metrics
}

func (c *Calculator) calculateClassMetrics(file *model.File, lines []string) []model.ClassMetrics {
	var metrics []model.ClassMetrics
	classMethodMap := make(map[string][]model.Call)

	for _, fn := range file.Functions {
		if fn.ParentClass != "" {
			classMethodMap[fn.ParentClass] = append(classMethodMap[fn.ParentClass], fn.Calls...)
		}
	}

	for _, cls := range file.Classes {
		classLines := lines[cls.StartLine-1 : cls.EndLine]
		loc := parser.CountEffectiveLOC(classLines, getCommentPrefix(file.Language)...)

		cbo := c.calculateCBO(cls, file)
		rfc := c.calculateRFC(cls, file, classMethodMap[cls.Name])

		metrics = append(metrics, model.ClassMetrics{
			FilePath:    file.Path,
			ClassName:   cls.Name,
			MethodCount: len(cls.Methods),
			AttrCount:   len(cls.Attributes),
			CBO:         cbo,
			RFC:         rfc,
			LOC:         loc,
		})
	}

	return metrics
}

func (c *Calculator) calculateCyclomaticComplexity(lines []string, lang model.Language) int {
	complexity := 1
	branchKeywords := getBranchKeywords(lang)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isCommentLine(trimmed, lang) {
			continue
		}

		for _, kw := range branchKeywords {
			if strings.HasPrefix(trimmed, kw) || strings.Contains(trimmed, " "+kw+" ") {
				complexity++
			}
		}

		if strings.Contains(trimmed, "&&") {
			complexity++
		}
		if strings.Contains(trimmed, "||") {
			complexity++
		}
		if strings.Contains(trimmed, "?") && strings.Contains(trimmed, ":") {
			complexity++
		}
	}

	return complexity
}

func (c *Calculator) calculateCognitiveComplexity(lines []string, lang model.Language) int {
	complexity := 0
	nestingLevel := 0
	branchKeywords := getBranchKeywords(lang)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isCommentLine(trimmed, lang) {
			continue
		}

		indentLevel := getLineIndent(line)

		isBranch := false
		for _, kw := range branchKeywords {
			if strings.HasPrefix(trimmed, kw) {
				isBranch = true
				break
			}
		}

		if strings.Contains(trimmed, "{") && indentLevel >= 0 {
			if isBranch {
				weight := nestingLevel + 1
				complexity += weight
				nestingLevel++
			}
		}

		if strings.Contains(trimmed, "}") && nestingLevel > 0 {
			nestingLevel--
		}
	}

	return complexity
}

func (c *Calculator) calculateCBO(cls model.Class, file *model.File) int {
	dependencies := make(map[string]bool)

	for _, imp := range file.Imports {
		dependencies[imp.Target] = true
	}

	for _, parent := range cls.Inherits {
		dependencies[parent] = true
	}

	return len(dependencies)
}

func (c *Calculator) calculateRFC(cls model.Class, file *model.File, calls []model.Call) int {
	rfc := len(cls.Methods)
	calledMethods := make(map[string]bool)

	for _, call := range calls {
		calledMethods[call.Callee] = true
	}

	rfc += len(calledMethods)
	return rfc
}

func (c *Calculator) checkViolations(file *model.File, funcMetrics []model.FunctionMetrics, classMetrics []model.ClassMetrics) []model.Violation {
	var violations []model.Violation
	thresholds := c.config.Thresholds

	for _, fm := range funcMetrics {
		if c.config.Rules["cyclomatic-complexity"] {
			if fm.CyclomaticComplexity >= thresholds.CyclomaticComplexityRed {
				violations = append(violations, model.Violation{
					RuleID:    "cyclomatic-complexity",
					Severity:  model.SeverityCritical,
					Message:   "圈复杂度过高",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
					Suggestion: "考虑重构，将复杂逻辑拆分为多个小函数",
				})
			} else if fm.CyclomaticComplexity >= thresholds.CyclomaticComplexityYellow {
				violations = append(violations, model.Violation{
					RuleID:    "cyclomatic-complexity",
					Severity:  model.SeverityMedium,
					Message:   "圈复杂度较高",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
				})
			}
		}

		if c.config.Rules["function-length"] {
			if fm.LOC >= thresholds.FunctionLOCRed {
				violations = append(violations, model.Violation{
					RuleID:    "function-length",
					Severity:  model.SeverityHigh,
					Message:   "函数过长",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
					Suggestion: "考虑重构，将长函数拆分为多个小函数",
				})
			} else if fm.LOC >= thresholds.FunctionLOCYellow {
				violations = append(violations, model.Violation{
					RuleID:    "function-length",
					Severity:  model.SeverityMedium,
					Message:   "函数较长",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
				})
			}
		}

		if c.config.Rules["long-parameter-list"] {
			if fm.ParamCount >= thresholds.ParamCountRed {
				violations = append(violations, model.Violation{
					RuleID:    "long-parameter-list",
					Severity:  model.SeverityHigh,
					Message:   "参数列表过长",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
					Suggestion: "考虑使用参数对象封装相关参数",
				})
			} else if fm.ParamCount >= thresholds.ParamCountYellow {
				violations = append(violations, model.Violation{
					RuleID:    "long-parameter-list",
					Severity:  model.SeverityMedium,
					Message:   "参数列表较长",
					FilePath:  fm.FilePath,
					StartLine: fm.StartLine,
					EndLine:   fm.EndLine,
				})
			}
		}
	}

	for _, cm := range classMetrics {
		if c.config.Rules["god-class"] {
			if cm.MethodCount >= thresholds.ClassMethodCountRed {
				violations = append(violations, model.Violation{
					RuleID:    "god-class",
					Severity:  model.SeverityHigh,
					Message:   "神类检测: 方法数量过多",
					FilePath:  cm.FilePath,
					StartLine: 1,
					EndLine:   cm.LOC,
					Suggestion: "考虑拆分类，将职责分离到多个类中",
				})
			} else if cm.MethodCount >= thresholds.ClassMethodCountYellow {
				violations = append(violations, model.Violation{
					RuleID:    "god-class",
					Severity:  model.SeverityMedium,
					Message:   "神类检测: 方法数量较多",
					FilePath:  cm.FilePath,
					StartLine: 1,
					EndLine:   cm.LOC,
				})
			}

			if cm.CBO >= thresholds.CBORed {
				violations = append(violations, model.Violation{
					RuleID:    "high-coupling",
					Severity:  model.SeverityHigh,
					Message:   "类耦合度过高",
					FilePath:  cm.FilePath,
					StartLine: 1,
					EndLine:   cm.LOC,
				})
			}
		}
	}

	return violations
}

func getCommentPrefix(lang model.Language) []string {
	switch lang {
	case model.LangPython:
		return []string{"#"}
	case model.LangGo, model.LangJavaScript, model.LangTypeScript, model.LangJava:
		return []string{"//", "/*", "*"}
	default:
		return []string{"//", "#"}
	}
}

func getBranchKeywords(lang model.Language) []string {
	switch lang {
	case model.LangPython:
		return []string{"if ", "elif ", "for ", "while ", "except ", "finally:", "match ", "case "}
	case model.LangGo:
		return []string{"if ", "else if ", "for ", "switch ", "case ", "select "}
	case model.LangJavaScript, model.LangTypeScript, model.LangJava:
		return []string{"if ", "else if ", "for ", "while ", "switch ", "case ", "catch "}
	default:
		return []string{"if ", "for ", "while ", "case ", "catch "}
	}
}

func isCommentLine(line string, lang model.Language) bool {
	prefixes := getCommentPrefix(lang)
	trimmed := strings.TrimSpace(line)
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func getLineIndent(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' || c == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

package parser

import (
	"regexp"
	"strings"

	"github.com/code-quality/cli/pkg/model"
)

type CStyleParser struct {
	lang      model.Language
	exts      []string
	funcRegex *regexp.Regexp
	classRegex *regexp.Regexp
}

func NewJavaScriptParser() *CStyleParser {
	return &CStyleParser{
		lang: model.LangJavaScript,
		exts: []string{".js", ".jsx"},
		funcRegex: regexp.MustCompile(
			`(?:(?:export\s+)?(?:default\s+)?(?:async\s+)?(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?function)`,
		),
		classRegex: regexp.MustCompile(`(?:export\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?\s*\{`),
	}
}

func NewTypeScriptParser() *CStyleParser {
	return &CStyleParser{
		lang: model.LangTypeScript,
		exts: []string{".ts", ".tsx"},
		funcRegex: regexp.MustCompile(
			`(?:(?:export\s+)?(?:default\s+)?(?:async\s+)?(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*[:=]\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>))`,
		),
		classRegex: regexp.MustCompile(`(?:export\s+)?(?:abstract\s+)?class\s+(\w+)(?:\s+(?:extends|implements)\s+([\w\s,]+))?\s*\{`),
	}
}

func NewJavaParser() *CStyleParser {
	return &CStyleParser{
		lang: model.LangJava,
		exts: []string{".java"},
		funcRegex: regexp.MustCompile(
			`(?:(?:public|private|protected|static|final|abstract|synchronized|native|strictfp)\s+)*[\w<>\[\]]+\s+(\w+)\s*\(`,
		),
		classRegex: regexp.MustCompile(
			`(?:(?:public|private|protected|static|final|abstract)\s+)*(?:class|interface|enum)\s+(\w+)(?:\s+(?:extends|implements)\s+([\w\s,<>\.]+))?\s*\{`,
		),
	}
}

func init() {
	Register(NewJavaScriptParser())
	Register(NewTypeScriptParser())
	Register(NewJavaParser())
}

func (p *CStyleParser) Language() model.Language {
	return p.lang
}

func (p *CStyleParser) Extensions() []string {
	return p.exts
}

func (p *CStyleParser) Parse(filePath string, content []byte) (*model.File, error) {
	lines := SplitLines(string(content))
	file := &model.File{
		Path:     filePath,
		Language: p.lang,
		Lines:    len(lines),
	}

	classes := p.extractClasses(lines)
	file.Classes = classes

	functions := p.extractFunctions(lines, classes)
	file.Functions = functions

	imports := p.extractImports(lines)
	file.Imports = imports

	p.extractCalls(lines, file)

	return file, nil
}

func (p *CStyleParser) extractFunctions(lines []string, classes []model.Class) []model.Function {
	var functions []model.Function
	var currentFunc *model.Function
	var braceDepth int

	for i, line := range lines {
		lineNum := i + 1

		if currentFunc == nil {
			if matches := p.funcRegex.FindStringSubmatch(line); matches != nil {
				funcName := matches[1]
				if funcName == "" && len(matches) > 2 {
					funcName = matches[2]
				}
				if funcName == "" || funcName == "if" || funcName == "for" || 
					funcName == "while" || funcName == "switch" || funcName == "catch" {
					continue
				}

				paramContent := extractMultiLineParams(lines, i)
				params := SplitParams(paramContent)

				currentFunc = &model.Function{
					Name:      funcName,
					StartLine: lineNum,
					Params:    params,
				}
				braceDepth = 0

				if strings.Contains(line, "{") {
					braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
				}
			}
			continue
		}

		for _, c := range line {
			if c == '{' {
				braceDepth++
			} else if c == '}' {
				braceDepth--
				if braceDepth == 0 {
					currentFunc.EndLine = lineNum
					currentFunc.NestingDepth = CalculateNestingDepth(
						lines, currentFunc.StartLine-1, currentFunc.EndLine-1, "{", "}")
					currentFunc.ParentClass = findParentClass(currentFunc.StartLine, classes)
					functions = append(functions, *currentFunc)
					currentFunc = nil
					break
				}
			}
		}
	}

	return functions
}

func (p *CStyleParser) extractClasses(lines []string) []model.Class {
	var classes []model.Class
	var currentClass *model.Class
	var braceDepth int
	methodSet := make(map[string]bool)
	attrSet := make(map[string]bool)

	for i, line := range lines {
		lineNum := i + 1

		if currentClass == nil {
			if matches := p.classRegex.FindStringSubmatch(line); matches != nil {
				className := matches[1]
				var inherits []string
				if len(matches) > 2 && matches[2] != "" {
					for _, inh := range strings.Split(matches[2], ",") {
						inh = strings.TrimSpace(inh)
						if inh != "" {
							inherits = append(inherits, inh)
						}
					}
				}

				currentClass = &model.Class{
					Name:      className,
					Inherits:  inherits,
					StartLine: lineNum,
				}
				methodSet = make(map[string]bool)
				attrSet = make(map[string]bool)
				braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && braceDepth == 1 {
			if p.isMethodDefinition(line) {
				if methodName := extractMethodName(line, p.lang); methodName != "" {
					methodSet[methodName] = true
				}
			} else if p.isAttributeDefinition(line) {
				if attrName := extractAttrName(line); attrName != "" {
					attrSet[attrName] = true
				}
			}
		}

		for _, c := range line {
			if c == '{' {
				braceDepth++
			} else if c == '}' {
				braceDepth--
				if braceDepth == 0 {
					currentClass.EndLine = lineNum
					for m := range methodSet {
						currentClass.Methods = append(currentClass.Methods, m)
					}
					for a := range attrSet {
						currentClass.Attributes = append(currentClass.Attributes, a)
					}
					classes = append(classes, *currentClass)
					currentClass = nil
					break
				}
			}
		}
	}

	return classes
}

func (p *CStyleParser) isMethodDefinition(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false
	}
	if strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")") {
		if strings.Contains(trimmed, "function") || 
			(p.lang == model.LangJava && regexp.MustCompile(`^\s*(?:public|private|protected)?\s+[\w<>\[\]]+\s+\w+\s*\(`).MatchString(line)) {
			return true
		}
		if (p.lang == model.LangJavaScript || p.lang == model.LangTypeScript) {
			if regexp.MustCompile(`^\s*(?:static\s+)?(?:async\s+)?\w+\s*\(`).MatchString(trimmed) {
				return !strings.HasPrefix(trimmed, "if") && 
					!strings.HasPrefix(trimmed, "for") && 
					!strings.HasPrefix(trimmed, "while") &&
					!strings.HasPrefix(trimmed, "switch") &&
					!strings.HasPrefix(trimmed, "catch")
			}
		}
	}
	return false
}

func (p *CStyleParser) isAttributeDefinition(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false
	}
	if strings.Contains(trimmed, "=") || strings.HasSuffix(trimmed, ";") {
		return true
	}
	return false
}

func extractMethodName(line string, lang model.Language) string {
	if lang == model.LangJava {
		re := regexp.MustCompile(`(?:public|private|protected|static|final|abstract)\s+[\w<>\[\]]+\s+(\w+)\s*\(`)
		if matches := re.FindStringSubmatch(line); matches != nil {
			return matches[1]
		}
		re2 := regexp.MustCompile(`[\w<>\[\]]+\s+(\w+)\s*\(`)
		if matches := re2.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			if name != "if" && name != "for" && name != "while" && name != "switch" {
				return name
			}
		}
	} else {
		re := regexp.MustCompile(`(?:static\s+)?(?:async\s+)?(\w+)\s*\(`)
		if matches := re.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			if name != "if" && name != "for" && name != "while" && name != "switch" && name != "catch" {
				return name
			}
		}
	}
	return ""
}

func extractAttrName(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.Contains(trimmed, "=") {
		parts := strings.SplitN(trimmed, "=", 2)
		left := strings.TrimSpace(parts[0])
		fields := strings.Fields(left)
		if len(fields) > 0 {
			name := fields[len(fields)-1]
			name = strings.Trim(name, ";")
			return name
		}
	}
	return ""
}

func (p *CStyleParser) extractImports(lines []string) []model.Import {
	var imports []model.Import
	importRegex := regexp.MustCompile(`^\s*(?:import|from)\s+['"]([^'"]+)['"]`)
	requireRegex := regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	javaImportRegex := regexp.MustCompile(`^\s*import\s+(.+);`)

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if p.lang == model.LangJava {
			if matches := javaImportRegex.FindStringSubmatch(line); matches != nil {
				imp := strings.TrimSpace(matches[1])
				imports = append(imports, model.Import{
					Source: ".",
					Target: imp,
					Line:   lineNum,
				})
			}
		} else {
			if matches := importRegex.FindStringSubmatch(line); matches != nil {
				imports = append(imports, model.Import{
					Source: ".",
					Target: matches[1],
					Line:   lineNum,
				})
				continue
			}
			if matches := requireRegex.FindStringSubmatch(line); matches != nil {
				imports = append(imports, model.Import{
					Source: ".",
					Target: matches[1],
					Line:   lineNum,
				})
			}
		}
	}

	return imports
}

func (p *CStyleParser) extractCalls(lines []string, file *model.File) {
	callRegex := regexp.MustCompile(`(\w+)\s*\(`)
	keywords := map[string]bool{
		"if": true, "for": true, "while": true, "switch": true,
		"catch": true, "function": true, "class": true, "return": true,
		"new": true, "typeof": true, "instanceof": true,
	}

	for fi := range file.Functions {
		fn := &file.Functions[fi]
		var calls []model.Call
		for i := fn.StartLine - 1; i < fn.EndLine && i < len(lines); i++ {
			line := lines[i]
			if matches := callRegex.FindAllStringSubmatch(line, -1); matches != nil {
				for _, m := range matches {
					callee := m[1]
					if !keywords[callee] {
						calls = append(calls, model.Call{
							Callee: callee,
							Line:   i + 1,
						})
					}
				}
			}
		}
		fn.Calls = calls
	}
}

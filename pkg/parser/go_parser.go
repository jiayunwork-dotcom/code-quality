package parser

import (
	"regexp"
	"strings"

	"github.com/code-quality/cli/pkg/model"
)

var (
	goFuncRegex  = regexp.MustCompile(`^(\s*)func\s+(\([^)]+\)\s*)?(\w+)\s*\(`)
	goStructRegex = regexp.MustCompile(`^(\s*)type\s+(\w+)\s+struct\s*\{`)
	goImportRegex = regexp.MustCompile(`^\s*import\s+(.+)$`)
	goCallRegex   = regexp.MustCompile(`(\w+)\s*\(`)
)

type GoParser struct{}

func init() {
	Register(&GoParser{})
}

func (p *GoParser) Language() model.Language {
	return model.LangGo
}

func (p *GoParser) Extensions() []string {
	return []string{".go"}
}

func (p *GoParser) Parse(filePath string, content []byte) (*model.File, error) {
	lines := SplitLines(string(content))
	file := &model.File{
		Path:     filePath,
		Language: model.LangGo,
		Lines:    len(lines),
	}

	functions, err := p.extractFunctions(lines)
	if err != nil {
		return nil, err
	}
	file.Functions = functions

	classes, err := p.extractClasses(lines)
	if err != nil {
		return nil, err
	}
	file.Classes = classes

	imports, err := p.extractImports(lines)
	if err != nil {
		return nil, err
	}
	file.Imports = imports

	p.extractCalls(lines, file)

	return file, nil
}

func (p *GoParser) extractFunctions(lines []string) ([]model.Function, error) {
	var functions []model.Function
	var currentFunc *model.Function
	var braceDepth int

	for i, line := range lines {
		lineNum := i + 1

		if matches := goFuncRegex.FindStringSubmatch(line); matches != nil && currentFunc == nil {
			funcName := matches[3]
			paramContent := ExtractBracketContent(line[matches[0]:len(line)], '(', ')')
			params := SplitParams(paramContent)

			currentFunc = &model.Function{
				Name:      funcName,
				StartLine: lineNum,
				Params:    params,
			}
			braceDepth = 0
		}

		if currentFunc != nil {
			for _, c := range line {
				if c == '{' {
					braceDepth++
				} else if c == '}' {
					braceDepth--
					if braceDepth == 0 {
						currentFunc.EndLine = lineNum
						currentFunc.NestingDepth = CalculateNestingDepth(
							lines, currentFunc.StartLine-1, currentFunc.EndLine-1, "{", "}")
						functions = append(functions, *currentFunc)
						currentFunc = nil
						break
					}
				}
			}
		}
	}

	return functions, nil
}

func (p *GoParser) extractClasses(lines []string) ([]model.Class, error) {
	var classes []model.Class
	var currentClass *model.Class
	var braceDepth int
	methodSet := make(map[string]bool)
	attrSet := make(map[string]bool)

	for i, line := range lines {
		lineNum := i + 1

		if matches := goStructRegex.FindStringSubmatch(line); matches != nil && currentClass == nil {
			currentClass = &model.Class{
				Name:      matches[2],
				StartLine: lineNum,
			}
			methodSet = make(map[string]bool)
			attrSet = make(map[string]bool)
			braceDepth = 0
		}

		if currentClass != nil {
			if currentClass.StartLine == lineNum {
				braceDepth = 1
				continue
			}

			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") && braceDepth == 1 {
				if strings.Contains(trimmed, "func(") {
					if methodName := extractGoMethodName(trimmed); methodName != "" {
						methodSet[methodName] = true
					}
				} else if !strings.HasPrefix(trimmed, "}") {
					parts := strings.Fields(trimmed)
					if len(parts) >= 1 && parts[0] != "" {
						attrName := strings.Trim(parts[0], "*")
						if !strings.HasPrefix(attrName, "//") && attrName != "" {
							attrSet[attrName] = true
						}
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
	}

	p.associateMethodsWithClasses(lines, &classes)

	return classes, nil
}

func extractGoMethodName(line string) string {
	idx := strings.Index(line, "func(")
	if idx == -1 {
		return ""
	}
	rest := line[idx+4:]
	idx = strings.Index(rest, ")")
	if idx == -1 {
		return ""
	}
	rest = strings.TrimSpace(rest[idx+1:])
	parts := strings.Fields(rest)
	if len(parts) > 0 {
		name := parts[0]
		if name != "{" {
			return name
		}
	}
	return ""
}

func (p *GoParser) associateMethodsWithClasses(lines []string, classes *[]model.Class) {
	for i, line := range lines {
		if matches := goFuncRegex.FindStringSubmatch(line); matches != nil {
			receiver := matches[2]
			if receiver != "" {
				receiver = strings.Trim(receiver, "()")
				parts := strings.Fields(receiver)
				if len(parts) >= 2 {
					className := strings.Trim(parts[1], "*")
					for ci := range *classes {
						if (*classes)[ci].Name == className {
							funcName := matches[3]
							exists := false
							for _, m := range (*classes)[ci].Methods {
								if m == funcName {
									exists = true
									break
								}
							}
							if !exists {
								(*classes)[ci].Methods = append((*classes)[ci].Methods, funcName)
							}
						}
					}
				}
			}
		}
	}
}

func (p *GoParser) extractImports(lines []string) ([]model.Import, error) {
	var imports []model.Import
	inBlock := false

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "import (" {
			inBlock = true
			continue
		}
		if inBlock {
			if trimmed == ")" {
				inBlock = false
				continue
			}
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				imp := strings.Trim(trimmed, "\"")
				imp = strings.Fields(imp)[len(strings.Fields(imp))-1]
				imp = strings.Trim(imp, "\"")
				imports = append(imports, model.Import{
					Source: ".",
					Target: imp,
					Line:   lineNum,
				})
			}
			continue
		}

		if matches := goImportRegex.FindStringSubmatch(line); matches != nil {
			impStr := strings.TrimSpace(matches[1])
			if strings.HasPrefix(impStr, "(") {
				inBlock = true
				continue
			}
			imp := strings.Trim(impStr, "\"")
			imports = append(imports, model.Import{
				Source: ".",
				Target: imp,
				Line:   lineNum,
			})
		}
	}

	return imports, nil
}

func (p *GoParser) extractCalls(lines []string, file *model.File) {
	for fi := range file.Functions {
		fn := &file.Functions[fi]
		var calls []model.Call
		for i := fn.StartLine - 1; i < fn.EndLine && i < len(lines); i++ {
			line := lines[i]
			if matches := goCallRegex.FindAllStringSubmatch(line, -1); matches != nil {
				for _, m := range matches {
					callee := m[1]
					if callee != "if" && callee != "for" && callee != "switch" && 
						callee != "func" && callee != "return" && callee != "go" {
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

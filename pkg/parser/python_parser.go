package parser

import (
	"regexp"
	"strings"

	"github.com/code-quality/cli/pkg/model"
)

var (
	pyFuncRegex    = regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(`)
	pyClassRegex   = regexp.MustCompile(`^(\s*)class\s+(\w+)\s*(\([^)]+\))?\s*:`)
	pyImportRegex1 = regexp.MustCompile(`^\s*import\s+(.+)$`)
	pyImportRegex2 = regexp.MustCompile(`^\s*from\s+(.+)\s+import\s+(.+)$`)
	pyCallRegex    = regexp.MustCompile(`(\w+)\s*\(`)
)

type PythonParser struct{}

func init() {
	Register(&PythonParser{})
}

func (p *PythonParser) Language() model.Language {
	return model.LangPython
}

func (p *PythonParser) Extensions() []string {
	return []string{".py"}
}

func (p *PythonParser) Parse(filePath string, content []byte) (*model.File, error) {
	lines := SplitLines(string(content))
	file := &model.File{
		Path:     filePath,
		Language: model.LangPython,
		Lines:    len(lines),
	}

	classes, err := p.extractClasses(lines)
	if err != nil {
		return nil, err
	}
	file.Classes = classes

	functions, err := p.extractFunctions(lines, classes)
	if err != nil {
		return nil, err
	}
	file.Functions = functions

	imports, err := p.extractImports(lines)
	if err != nil {
		return nil, err
	}
	file.Imports = imports

	p.extractCalls(lines, file)

	return file, nil
}

func (p *PythonParser) extractFunctions(lines []string, classes []model.Class) ([]model.Function, error) {
	var functions []model.Function
	var currentFunc *model.Function
	var currentIndent int

	for i, line := range lines {
		lineNum := i + 1
		indent := getIndentLevel(line)

		if matches := pyFuncRegex.FindStringSubmatch(line); matches != nil {
			if currentFunc != nil {
				currentFunc.EndLine = lineNum - 1
				currentFunc.NestingDepth = p.calculatePythonNesting(lines, currentFunc.StartLine-1, currentFunc.EndLine-1)
				currentFunc.ParentClass = findParentClass(currentFunc.StartLine, classes)
				functions = append(functions, *currentFunc)
			}

			funcName := matches[2]
			paramContent := ExtractBracketContent(line, '(', ')')
			params := SplitParams(paramContent)

			currentFunc = &model.Function{
				Name:      funcName,
				StartLine: lineNum,
				Params:    params,
			}
			currentIndent = indent
			continue
		}

		if currentFunc != nil && line != "" && indent <= currentIndent && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			if !strings.HasSuffix(strings.TrimSpace(line), "\\") {
				currentFunc.EndLine = lineNum - 1
				currentFunc.NestingDepth = p.calculatePythonNesting(lines, currentFunc.StartLine-1, currentFunc.EndLine-1)
				currentFunc.ParentClass = findParentClass(currentFunc.StartLine, classes)
				functions = append(functions, *currentFunc)
				currentFunc = nil
			}
		}
	}

	if currentFunc != nil {
		currentFunc.EndLine = len(lines)
		currentFunc.NestingDepth = p.calculatePythonNesting(lines, currentFunc.StartLine-1, currentFunc.EndLine-1)
		currentFunc.ParentClass = findParentClass(currentFunc.StartLine, classes)
		functions = append(functions, *currentFunc)
	}

	return functions, nil
}

func findParentClass(lineNum int, classes []model.Class) string {
	for _, cls := range classes {
		if lineNum >= cls.StartLine && lineNum <= cls.EndLine {
			return cls.Name
		}
	}
	return ""
}

func (p *PythonParser) extractClasses(lines []string) ([]model.Class, error) {
	var classes []model.Class
	var currentClass *model.Class
	var classIndent int
	methodSet := make(map[string]bool)
	attrSet := make(map[string]bool)

	for i, line := range lines {
		lineNum := i + 1
		indent := getIndentLevel(line)
		trimmed := strings.TrimSpace(line)

		if matches := pyClassRegex.FindStringSubmatch(line); matches != nil {
			if currentClass != nil {
				currentClass.EndLine = lineNum - 1
				for m := range methodSet {
					currentClass.Methods = append(currentClass.Methods, m)
				}
				for a := range attrSet {
					currentClass.Attributes = append(currentClass.Attributes, a)
				}
				classes = append(classes, *currentClass)
			}

			className := matches[2]
			inherits := []string{}
			if matches[3] != "" {
				inheritStr := strings.Trim(matches[3], "()")
				for _, inh := range strings.Split(inheritStr, ",") {
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
			classIndent = indent
			methodSet = make(map[string]bool)
			attrSet = make(map[string]bool)
			continue
		}

		if currentClass != nil {
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}

			if indent > classIndent {
				if matches := pyFuncRegex.FindStringSubmatch(line); matches != nil {
					methodSet[matches[2]] = true
				} else if strings.Contains(trimmed, "self.") && !strings.Contains(trimmed, "def ") {
					attrMatches := regexp.MustCompile(`self\.(\w+)\s*=`).FindAllStringSubmatch(line, -1)
					for _, m := range attrMatches {
						attrSet[m[1]] = true
					}
				}
			} else if indent <= classIndent && !strings.HasPrefix(trimmed, "#") {
				currentClass.EndLine = lineNum - 1
				for m := range methodSet {
					currentClass.Methods = append(currentClass.Methods, m)
				}
				for a := range attrSet {
					currentClass.Attributes = append(currentClass.Attributes, a)
				}
				classes = append(classes, *currentClass)
				currentClass = nil
			}
		}
	}

	if currentClass != nil {
		currentClass.EndLine = len(lines)
		for m := range methodSet {
			currentClass.Methods = append(currentClass.Methods, m)
		}
		for a := range attrSet {
			currentClass.Attributes = append(currentClass.Attributes, a)
		}
		classes = append(classes, *currentClass)
	}

	p.associateFunctionsWithClasses(&classes, &lines)

	return classes, nil
}

func (p *PythonParser) associateFunctionsWithClasses(classes *[]model.Class, lines *[]string) {
	for ci := range *classes {
		class := &(*classes)[ci]
		for _, method := range class.Methods {
			for fi := range (*classes) {
				if fi != ci {
					for _, m := range (*classes)[fi].Methods {
						if m == method {
							continue
						}
					}
				}
			}
		}
	}
}

func (p *PythonParser) extractImports(lines []string) ([]model.Import, error) {
	var imports []model.Import

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if matches := pyImportRegex2.FindStringSubmatch(line); matches != nil {
			source := strings.TrimSpace(matches[1])
			imports = append(imports, model.Import{
				Source: ".",
				Target: source,
				Line:   lineNum,
			})
			continue
		}

		if matches := pyImportRegex1.FindStringSubmatch(line); matches != nil {
			impStr := matches[1]
			for _, imp := range strings.Split(impStr, ",") {
				imp = strings.TrimSpace(imp)
				imp = strings.Split(imp, " ")[0]
				if imp != "" {
					imports = append(imports, model.Import{
						Source: ".",
						Target: imp,
						Line:   lineNum,
					})
				}
			}
		}
	}

	return imports, nil
}

func (p *PythonParser) extractCalls(lines []string, file *model.File) {
	for fi := range file.Functions {
		fn := &file.Functions[fi]
		var calls []model.Call
		for i := fn.StartLine - 1; i < fn.EndLine && i < len(lines); i++ {
			line := lines[i]
			if matches := pyCallRegex.FindAllStringSubmatch(line, -1); matches != nil {
				for _, m := range matches {
					callee := m[1]
					if callee != "if" && callee != "for" && callee != "while" &&
						callee != "def" && callee != "class" && callee != "elif" &&
						callee != "except" && callee != "print" {
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

func (p *PythonParser) calculatePythonNesting(lines []string, start, end int) int {
	keywords := []string{"if ", "elif ", "else:", "for ", "while ", "try:", "except ", "finally:", "with "}
	maxDepth := 0
	stack := make([]int, 0)

	for i := start; i <= end && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		indent := getIndentLevel(lines[i])

		for len(stack) > 0 && stack[len(stack)-1] >= indent {
			stack = stack[:len(stack)-1]
		}

		isBlock := false
		for _, kw := range keywords {
			if strings.HasPrefix(line, kw) {
				isBlock = true
				break
			}
		}

		if isBlock || strings.HasSuffix(line, ":") {
			stack = append(stack, indent)
		}

		if len(stack) > maxDepth {
			maxDepth = len(stack)
		}
	}

	return maxDepth
}

func getIndentLevel(line string) int {
	count := 0
	for _, c := range line {
		if c == ' ' {
			count++
		} else if c == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

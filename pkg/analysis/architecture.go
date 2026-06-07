package analysis

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/code-quality/cli/pkg/model"
)

type ArchitectureAnalyzer struct {
	config *model.Config
}

func NewArchitectureAnalyzer(config *model.Config) *ArchitectureAnalyzer {
	return &ArchitectureAnalyzer{
		config: config,
	}
}

func (a *ArchitectureAnalyzer) Analyze(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue

	if a.config.Rules["cyclic-dependencies"] {
		issues = append(issues, a.detectCyclicDependencies(files)...)
	}

	if a.config.Rules["layer-violations"] {
		issues = append(issues, a.detectLayerViolations(files)...)
	}

	if a.config.Rules["god-class"] {
		issues = append(issues, a.detectGodClasses(files)...)
	}

	if a.config.Rules["feature-envy"] {
		issues = append(issues, a.detectFeatureEnvy(files)...)
	}

	if a.config.Rules["data-clumps"] {
		issues = append(issues, a.detectDataClumps(files)...)
	}

	return issues
}

func (a *ArchitectureAnalyzer) detectCyclicDependencies(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue

	graph := make(map[string][]string)
	nodeMap := make(map[string]bool)

	for _, fr := range files {
		if fr.File == nil {
			continue
		}
		src := filepath.Base(fr.File.Path)
		nodeMap[src] = true
		for _, imp := range fr.File.Imports {
			target := filepath.Base(imp.Target)
			if target == "" || target == "." {
				continue
			}
			graph[src] = append(graph[src], target)
			nodeMap[target] = true
		}
	}

	nodes := make([]string, 0, len(nodeMap))
	for n := range nodeMap {
		nodes = append(nodes, n)
	}

	sccs := tarjanSCC(graph, nodes)

	for _, scc := range sccs {
		if len(scc) >= 2 {
			sort.Strings(scc)
			issues = append(issues, model.ArchitectureIssue{
				Type:     "cyclic-dependency",
				Severity: model.SeverityHigh,
				Message:  "检测到循环依赖",
				Details:  scc,
			})
		}
	}

	return issues
}

func tarjanSCC(graph map[string][]string, nodes []string) [][]string {
	index := 0
	stack := make([]string, 0)
	onStack := make(map[string]bool)
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	var sccs [][]string

	var strongconnect func(v string)
	strongconnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range graph[v] {
			if _, ok := indices[w]; !ok {
				strongconnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		if lowlink[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	for _, n := range nodes {
		if _, ok := indices[n]; !ok {
			strongconnect(n)
		}
	}

	return sccs
}

func (a *ArchitectureAnalyzer) detectLayerViolations(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue

	if len(a.config.Layers) == 0 {
		return issues
	}

	layerMap := make(map[string]string)
	for _, layer := range a.config.Layers {
		for _, path := range layer.Paths {
			layerMap[path] = layer.Name
		}
	}

	allowedDeps := make(map[string]map[string]bool)
	for _, layer := range a.config.Layers {
		allowedDeps[layer.Name] = make(map[string]bool)
		for _, dep := range layer.MayDepend {
			allowedDeps[layer.Name][dep] = true
		}
		allowedDeps[layer.Name][layer.Name] = true
	}

	for _, fr := range files {
		if fr.File == nil {
			continue
		}

		srcLayer := a.getFileLayer(fr.File.Path, layerMap)
		if srcLayer == "" {
			continue
		}

		for _, imp := range fr.File.Imports {
			targetLayer := a.getFileLayer(imp.Target, layerMap)
			if targetLayer == "" {
				continue
			}

			if !allowedDeps[srcLayer][targetLayer] {
				issues = append(issues, model.ArchitectureIssue{
					Type:     "layer-violation",
					Severity: model.SeverityHigh,
					Message:  "层级违反: " + srcLayer + " 依赖了不允许的层 " + targetLayer,
					FilePath: fr.File.Path,
					Details:  []string{srcLayer + " -> " + targetLayer},
				})
			}
		}
	}

	return issues
}

func (a *ArchitectureAnalyzer) getFileLayer(path string, layerMap map[string]string) string {
	for pattern, layer := range layerMap {
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return layer
		}
		if strings.Contains(path, pattern) {
			return layer
		}
	}
	return ""
}

func (a *ArchitectureAnalyzer) detectGodClasses(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue
	threshold := a.config.Thresholds.ClassMethodCountYellow

	for _, fr := range files {
		for _, cm := range fr.Classes {
			if cm.MethodCount > threshold && cm.CBO > a.config.Thresholds.CBOYellow {
				issues = append(issues, model.ArchitectureIssue{
					Type:     "god-class",
					Severity: model.SeverityMedium,
					Message:  "神类检测: " + cm.ClassName + " 方法数量过多且耦合度高",
					FilePath: cm.FilePath,
					Details:  []string{"方法数: " + strconv.Itoa(cm.MethodCount), "耦合度: " + strconv.Itoa(cm.CBO)},
				})
			}
		}
	}

	return issues
}

func (a *ArchitectureAnalyzer) detectFeatureEnvy(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue

	selfAccessRegex := regexp.MustCompile(`(?:self|this)\.(\w+)`)
	objAccessRegex := regexp.MustCompile(`(\w+)\.(\w+)`)

	fileLinesCache := make(map[string][]string)

	for _, fr := range files {
		if fr.File == nil {
			continue
		}

		filePath := fr.File.Path
		lines, ok := fileLinesCache[filePath]
		if !ok {
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			lines = strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
			fileLinesCache[filePath] = lines
		}

		for _, cls := range fr.File.Classes {
			for _, fn := range fr.File.Functions {
				if fn.ParentClass != cls.Name {
					continue
				}

				selfAccessCount := 0
				otherObjAccess := make(map[string]int)
				excludeVars := map[string]bool{
					"self": true, "this": true, "True": true, "False": true,
					"None": true, "null": true, "undefined": true,
					"if": true, "for": true, "while": true, "else": true,
					"elif": true, "return": true, "print": true, "len": true,
					"str": true, "int": true, "float": true, "list": true,
					"dict": true, "set": true, "bool": true,
				}

				startIdx := fn.StartLine - 1
				endIdx := fn.EndLine
				if endIdx > len(lines) {
					endIdx = len(lines)
				}

				for i := startIdx; i < endIdx; i++ {
					line := lines[i]
					trimmed := strings.TrimSpace(line)
					if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
						continue
					}

					if matches := selfAccessRegex.FindAllStringSubmatch(line, -1); matches != nil {
						selfAccessCount += len(matches)
					}

					if matches := objAccessRegex.FindAllStringSubmatch(line, -1); matches != nil {
						for _, m := range matches {
							objName := m[1]
							if excludeVars[objName] {
								continue
							}
							if objName == cls.Name {
								continue
							}
							otherObjAccess[objName]++
						}
					}
				}

				for objName, count := range otherObjAccess {
					if count >= 4 && count > selfAccessCount {
						issues = append(issues, model.ArchitectureIssue{
							Type:     "feature-envy",
							Severity: model.SeverityMedium,
							Message:  "特征依恋: 方法 " + cls.Name + "." + fn.Name + " 大量访问对象 " + objName + " 的属性/方法",
							FilePath: filePath,
							Details:  []string{"访问外部对象次数: " + strconv.Itoa(count), "访问自身(self)次数: " + strconv.Itoa(selfAccessCount)},
						})
						break
					}
				}
			}
		}
	}

	return issues
}

func (a *ArchitectureAnalyzer) detectDataClumps(files []*model.FileReport) []model.ArchitectureIssue {
	var issues []model.ArchitectureIssue

	paramCombinations := make(map[string]int)
	paramFuncMap := make(map[string][]string)

	for _, fr := range files {
		if fr.File == nil {
			continue
		}
		for _, fn := range fr.File.Functions {
			if len(fn.Params) >= 3 {
				params := make([]string, len(fn.Params))
				copy(params, fn.Params)
				sort.Strings(params)
				key := strings.Join(params, ",")
				paramCombinations[key]++
				paramFuncMap[key] = append(paramFuncMap[key], fn.Name+"@"+fr.File.Path)
			}
		}
	}

	for key, count := range paramCombinations {
		if count >= 3 {
			params := strings.Split(key, ",")
			if len(params) >= 3 {
				issues = append(issues, model.ArchitectureIssue{
					Type:     "data-clumps",
					Severity: model.SeverityLow,
					Message:  "数据泥团检测: 参数组在多个函数中重复出现",
					Details:  append([]string{"参数: " + key}, paramFuncMap[key]...),
				})
			}
		}
	}

	return issues
}

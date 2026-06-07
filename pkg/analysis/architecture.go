package analysis

import (
	"path/filepath"
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

	allAttrs := make(map[string]map[string]bool)
	classAttrCount := make(map[string]int)
	for _, fr := range files {
		if fr.File == nil {
			continue
		}
		for _, cls := range fr.File.Classes {
			key := fr.File.Path + ":" + cls.Name
			allAttrs[key] = make(map[string]bool)
			for _, attr := range cls.Attributes {
				allAttrs[key][attr] = true
			}
			classAttrCount[key] = len(cls.Attributes)
		}
	}

	for _, fr := range files {
		if fr.File == nil {
			continue
		}
		for _, cls := range fr.File.Classes {
			selfKey := fr.File.Path + ":" + cls.Name
			selfAttrs := allAttrs[selfKey]

			for _, fn := range fr.File.Functions {
				if fn.ParentClass != cls.Name && fn.ParentClass != "" {
					continue
				}
				if fn.ParentClass == "" && !strings.HasPrefix(fn.Name, cls.Name) {
					continue
				}

				otherAccess := make(map[string]int)
				selfAccessCount := 0

				if len(fn.Calls) == 0 {
					continue
				}

				for _, call := range fn.Calls {
					if call.TargetObj != "" && call.TargetObj != "self" && call.TargetObj != "this" {
						for otherKey, attrs := range allAttrs {
							if otherKey == selfKey {
								continue
							}
							for attr := range attrs {
								if strings.Contains(call.Callee, attr) || strings.Contains(call.TargetObj, attr) {
									otherAccess[otherKey]++
								}
							}
						}
					} else if call.TargetObj == "self" || call.TargetObj == "this" {
						for selfAttr := range selfAttrs {
							if strings.Contains(call.Callee, selfAttr) {
								selfAccessCount++
							}
						}
					}
				}

				for otherKey, count := range otherAccess {
					if count > 3 && count > selfAccessCount {
						otherClassName := strings.SplitN(otherKey, ":", 2)[1]
						issues = append(issues, model.ArchitectureIssue{
							Type:     "feature-envy",
							Severity: model.SeverityMedium,
							Message:  "特征依恋: 方法 " + fn.Name + " 大量访问类 " + otherClassName + " 的属性",
							FilePath: fr.File.Path,
							Details:  []string{"访问外部属性次数: " + strconv.Itoa(count), "访问自身属性次数: " + strconv.Itoa(selfAccessCount)},
						})
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

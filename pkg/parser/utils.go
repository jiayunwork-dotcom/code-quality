package parser

import (
	"regexp"
	"strings"
)

var (
	blankLineRegex = regexp.MustCompile(`^\s*$`)
	commentLineRegex = map[string]*regexp.Regexp{
		"//":  regexp.MustCompile(`^\s*//`),
		"#":   regexp.MustCompile(`^\s*#`),
		"/*":  regexp.MustCompile(`^\s*/\*`),
	}
)

func CountEffectiveLOC(lines []string, commentPrefixes ...string) int {
	count := 0
	inBlockComment := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlockComment = true
			}
			continue
		}
		isComment := false
		for _, prefix := range commentPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				isComment = true
				break
			}
		}
		if isComment {
			continue
		}
		count++
	}
	return count
}

func SplitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func HasIgnoreComment(line string, ruleID string) bool {
	patterns := []string{
		"quality:ignore",
		"quality:ignore-next-line",
	}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			if ruleID == "" {
				return true
			}
			if strings.Contains(line, ruleID) {
				return true
			}
		}
	}
	return false
}

func ExtractBracketContent(s string, open, close rune) string {
	depth := 0
	start := -1
	for i, c := range s {
		if c == open {
			if depth == 0 {
				start = i
			}
			depth++
		} else if c == close {
			depth--
			if depth == 0 && start != -1 {
				return s[start+1 : i]
			}
		}
	}
	return ""
}

func SplitParams(paramStr string) []string {
	if paramStr == "" {
		return nil
	}
	var params []string
	var current strings.Builder
	depth := 0
	for _, c := range paramStr {
		switch c {
		case '(':
			depth++
			current.WriteRune(c)
		case ')':
			depth--
			current.WriteRune(c)
		case ',':
			if depth == 0 {
				p := strings.TrimSpace(current.String())
				if p != "" {
					params = append(params, p)
				}
				current.Reset()
			} else {
				current.WriteRune(c)
			}
		default:
			current.WriteRune(c)
		}
	}
	p := strings.TrimSpace(current.String())
	if p != "" {
		params = append(params, p)
	}
	return params
}

func CalculateNestingDepth(lines []string, startLine, endLine int, openBrackets, closeBrackets string) int {
	maxDepth := 0
	currentDepth := 0
	for i := startLine; i <= endLine && i < len(lines); i++ {
		line := lines[i]
		for _, c := range line {
			if strings.ContainsRune(openBrackets, c) {
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			} else if strings.ContainsRune(closeBrackets, c) {
				if currentDepth > 0 {
					currentDepth--
				}
			}
		}
	}
	return maxDepth
}

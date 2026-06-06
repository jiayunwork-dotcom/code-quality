package parser

import (
	"path/filepath"
	"strings"

	"github.com/code-quality/cli/pkg/model"
)

type Parser interface {
	Parse(filePath string, content []byte) (*model.File, error)
	Language() model.Language
	Extensions() []string
}

var parsers = map[model.Language]Parser{}

func Register(p Parser) {
	parsers[p.Language()] = p
}

func Get(lang model.Language) Parser {
	return parsers[lang]
}

func DetectLanguage(filePath string) model.Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".py":
		return model.LangPython
	case ".js":
		return model.LangJavaScript
	case ".ts", ".tsx":
		return model.LangTypeScript
	case ".java":
		return model.LangJava
	case ".go":
		return model.LangGo
	default:
		return ""
	}
}

func ParseFile(filePath string, content []byte) (*model.File, error) {
	lang := DetectLanguage(filePath)
	if lang == "" {
		return nil, nil
	}
	p := Get(lang)
	if p == nil {
		return nil, nil
	}
	return p.Parse(filePath, content)
}

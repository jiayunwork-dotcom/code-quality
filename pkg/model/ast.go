package model

import "encoding/json"

type Language string

const (
	LangPython     Language = "python"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangJava       Language = "java"
	LangGo         Language = "go"
)

type File struct {
	Path      string     `json:"path"`
	Language  Language   `json:"language"`
	Functions []Function `json:"functions"`
	Classes   []Class    `json:"classes"`
	Imports   []Import   `json:"imports"`
	Lines     int        `json:"lines"`
}

type Function struct {
	Name         string   `json:"name"`
	StartLine    int      `json:"start_line"`
	EndLine      int      `json:"end_line"`
	Params       []string `json:"params"`
	NestingDepth int      `json:"nesting_depth"`
	ParentClass  string   `json:"parent_class,omitempty"`
	Calls        []Call   `json:"calls,omitempty"`
}

type Class struct {
	Name        string   `json:"name"`
	Methods     []string `json:"methods"`
	Attributes  []string `json:"attributes"`
	Inherits    []string `json:"inherits,omitempty"`
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
}

type Import struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Line    int    `json:"line"`
}

type Call struct {
	Callee    string `json:"callee"`
	TargetObj string `json:"target_obj,omitempty"`
	Line      int    `json:"line"`
}

func (f *File) JSON() string {
	b, _ := json.MarshalIndent(f, "", "  ")
	return string(b)
}

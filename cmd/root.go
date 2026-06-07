package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/code-quality/cli/pkg/report"
	"github.com/code-quality/cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "code-quality [path]",
		Short: "源代码质量度量和架构腐化检测工具",
		Long: `一个功能强大的代码质量分析工具，支持多语言解析、复杂度度量、
架构腐化检测、Git变更热点分析等功能。`,
		Args: cobra.MaximumNArgs(1),
		RunE: runScan,
	}

	outputFormat string
	outputFile   string
	sinceDays    int
	diffRange    string
	withBaseline bool
	saveBaseline bool
	failOn       string
	incremental  bool
)

func init() {
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "输出格式: text, json, html, sarif")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "-", "输出文件路径, - 表示标准输出")
	rootCmd.Flags().IntVar(&sinceDays, "since", 0, "分析Git历史的时间范围(天数), 0 表示全部历史")
	rootCmd.Flags().StringVar(&diffRange, "diff", "", "Git diff范围, 如 HEAD~1, 只分析变更文件")
	rootCmd.Flags().BoolVar(&withBaseline, "baseline", false, "与基线对比")
	rootCmd.Flags().BoolVar(&saveBaseline, "save-baseline", false, "保存当前结果为基线")
	rootCmd.Flags().StringVar(&failOn, "fail-on", "critical,high", "导致退出码为1的严重级别: critical,high,medium,all")
	rootCmd.Flags().BoolVar(&incremental, "incremental", false, "增量分析模式，只分析最近一次提交修改的文件并与基线对比")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		projectRoot, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}
	}

	info, err := os.Stat(projectRoot)
	if err != nil {
		return fmt.Errorf("路径不存在: %s", projectRoot)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径必须是目录: %s", projectRoot)
	}

	options := scanner.ScanOptions{
		SinceDays:    sinceDays,
		DiffRange:    diffRange,
		WithBaseline: withBaseline,
		SaveBaseline: saveBaseline,
		FailOn:       failOn,
		Incremental:  incremental,
	}

	s, err := scanner.NewScanner(projectRoot, options)
	if err != nil {
		return err
	}

	result, err := s.Scan()
	if err != nil {
		return err
	}

	format := report.Format(outputFormat)
	reporter := report.NewReporter(format, outputFile)
	if err := reporter.Generate(result); err != nil {
		return err
	}

	exitCode := s.GetExitCode(result)
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

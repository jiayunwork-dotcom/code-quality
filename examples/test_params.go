package examples

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

// 测试多行参数提取
type UserService struct {
	db *sql.DB
}

// 多行参数的函数 - 用于测试参数提取
func (s *UserService) ProcessUserRequest(
	ctx context.Context,
	userID int64,
	requestData map[string]interface{},
	options map[string]bool,
	timeout time.Duration,
	maxRetries int,
	logger *Logger,
	metrics *MetricsCollector,
) (*UserResponse, error) {
	return nil, nil
}

// 包含复杂类型参数的函数
func (s *UserService) ComplexFunction(
	users []*User,
	filters map[string][]string,
	results chan<- *ProcessResult,
	handler func(user *User) error,
	w http.ResponseWriter,
	r *http.Request,
) error {
	return nil
}

// 单行多参数函数
func SingleLineFunc(a int, b string, c bool, d float64, e []int, f map[string]int) string {
	return "ok"
}

type Logger struct{}
type MetricsCollector struct{}
type User struct{}
type UserResponse struct{}
type ProcessResult struct{}

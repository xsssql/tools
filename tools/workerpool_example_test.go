package tools

// Package workerpool_test 包含 WorkerPool 的使用示例，覆盖以下场景：
//   - 示例1：简单爬虫任务（同类型结构体，string 结果）
//   - 示例2：多种不同结构体任务共用同一个 Pool（union 结果类型）
//   - 示例3：Fire-and-forget 模式 + 动态扩容 + 优雅停止
//   - 示例4：超时与重试联动演示

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// =============================================================================
// 示例 1：简单爬虫任务（string 结果类型）
// =============================================================================

// ScrapeTask 模拟一个 HTTP 抓取任务，返回抓取到的页面内容（string）。
type ScrapeTask struct {
	id  string // 任务唯一标识
	URL string // 目标 URL
}

func (t *ScrapeTask) TaskID() string { return t.id }

// Run 模拟 HTTP 请求，随机等待 0~200ms 模拟网络延迟。
// ctx 用于快速中断。
func (t *ScrapeTask) Run(ctx context.Context) (string, error) {
	select {
	case <-time.After(time.Duration(rand.Intn(200)) * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return fmt.Sprintf("已抓取: %s", t.URL), nil
}

// Example_simpleScraper 演示最基础的用法
func Example_simpleScraper() {
	// 创建一个处理 string 结果的 Pool
	pool := NewPool[string](Options{
		Workers:     3,                      // 初始启动 3 个 Worker goroutine
		MaxWorkers:  8,                      // 最多自动扩容到 8 个
		TaskTimeout: 5 * time.Second,        // 每个任务最多运行 5 秒
		MaxRetries:  2,                      // 失败后最多重试 2 次（共执行 3 次）
		RetryDelay:  200 * time.Millisecond, // 首次重试等 200ms，后续翻倍
		RateLimit:   10,                     // 每秒最多启动 10 个任务（间隔 100ms）
	})
	defer pool.StopGraceful()

	tasks := []Task[string]{
		&ScrapeTask{id: "t1", URL: "https://example.com/a"},
		&ScrapeTask{id: "t2", URL: "https://example.com/b"},
		&ScrapeTask{id: "t3", URL: "https://example.com/c"},
	}

	results := pool.SubmitAndCollect(tasks)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("[%s] 失败: %v\n", r.TaskID, r.Err)
		} else {
			fmt.Printf("[%s] 成功: %s\n", r.TaskID, r.Value)
		}
	}

	// Output:
	// [t1] 成功: 已抓取: https://example.com/a
	// [t2] 成功: 已抓取: https://example.com/b
	// [t3] 成功: 已抓取: https://example.com/c
}

// =============================================================================
// 示例 2：多种不同结构体任务共用同一个 Pool
// =============================================================================

type ResultPayload struct {
	Kind string
	Data any
}

type EmailTask struct {
	To      string
	Subject string
	Body    string
}

func (e *EmailTask) TaskID() string { return "email:" + e.To }
func (e *EmailTask) Run(_ context.Context) (ResultPayload, error) {
	time.Sleep(50 * time.Millisecond)
	return ResultPayload{Kind: "email", Data: "已发送至 " + e.To}, nil
}

type ReportTask struct {
	ReportName string
	Pages      int
}

func (r *ReportTask) TaskID() string { return "report:" + r.ReportName }
func (r *ReportTask) Run(_ context.Context) (ResultPayload, error) {
	time.Sleep(time.Duration(r.Pages*10) * time.Millisecond)
	return ResultPayload{Kind: "report", Data: fmt.Sprintf("%s（共 %d 页）", r.ReportName, r.Pages)}, nil
}

func Example_heterogeneousTasks() {
	pool := NewPool[ResultPayload](Options{
		Workers:    4,
		MaxWorkers: 10,
		MaxRetries: 1,
		RetryDelay: 100 * time.Millisecond,
		OnSuccess: func(id string, d time.Duration) {
			fmt.Printf("✓ %s 完成\n", id)
		},
		OnFailure: func(id string, err error, attempts int) {
			fmt.Printf("✗ %s 失败: %v\n", id, err)
		},
	})
	defer pool.StopGraceful()

	tasks := []Task[ResultPayload]{
		&EmailTask{To: "alice@example.com"},
		&EmailTask{To: "bob@example.com"},
		&ReportTask{ReportName: "Q1-2025", Pages: 5},
		&ReportTask{ReportName: "年度报告", Pages: 20},
	}

	results := pool.SubmitAndCollect(tasks)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("[%s] 错误: %v\n", r.TaskID, r.Err)
		} else {
			fmt.Printf("[%s] 结果: %v\n", r.TaskID, r.Value.Data)
		}
	}
}

// =============================================================================
// 示例 3：Fire-and-forget + 动态扩容 + 优雅停止
// =============================================================================

type ComputeTask struct {
	n int
}

func (c *ComputeTask) TaskID() string { return fmt.Sprintf("compute-%d", c.n) }
func (c *ComputeTask) Run(_ context.Context) (int, error) {
	time.Sleep(time.Duration(c.n) * time.Millisecond)
	return c.n * c.n, nil
}

func Example_dynamicScaleAndStop() {
	pool := NewPool[int](Options{
		Workers:    2,
		MaxWorkers: 16,
		QueueSize:  100,
	})

	for i := 1; i <= 50; i++ {
		_ = pool.Submit(&ComputeTask{n: i})
	}

	fmt.Println("提交后指标:", pool.Metrics())
	_ = pool.Scale(8)
	fmt.Println("扩容后指标:", pool.Metrics())

	pool.StopGraceful()
	fmt.Println("全部完成，最终指标:", pool.Metrics())
}

// =============================================================================
// 示例 4：超时与重试联动演示
// =============================================================================

type FlakeyTask struct {
	id       string
	failFor  int
	attempts int
}

func (f *FlakeyTask) TaskID() string { return f.id }
func (f *FlakeyTask) Run(_ context.Context) (string, error) {
	f.attempts++
	if f.attempts <= f.failFor {
		return "", fmt.Errorf("临时错误（第 %d 次尝试）", f.attempts)
	}
	return fmt.Sprintf("第 %d 次尝试成功", f.attempts), nil
}

func Example_retryAndTimeout() {
	pool := NewPool[string](Options{
		Workers:     2,
		TaskTimeout: 2 * time.Second,
		MaxRetries:  3,
		RetryDelay:  50 * time.Millisecond,
	})
	defer pool.StopGraceful()

	tasks := []Task[string]{
		&FlakeyTask{id: "flakey-1", failFor: 2},
		&FlakeyTask{id: "flakey-2", failFor: 5},
	}

	for _, r := range pool.SubmitAndCollect(tasks) {
		if r.Err != nil {
			fmt.Printf("[%s] 最终失败: %v\n", r.TaskID, r.Err)
		} else {
			fmt.Printf("[%s] 最终成功: %s\n", r.TaskID, r.Value)
		}
	}
}

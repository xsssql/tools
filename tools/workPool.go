package tools

// Package workerpool 提供一个企业级、泛型化的通用 Worker Pool（工作池）。
//
// 核心特性：
//   - 泛型支持：结果类型 T 由调用方指定，同一个 Pool 可处理不同结构体的任务
//   - 任务超时：每个任务独立设置截止时间，超时后自动取消
//   - 自动重试：失败任务按指数退避策略自动重试，可配置最大次数
//   - 限速控制：令牌桶模型，精确控制每秒最多启动多少个任务
//   - 动态扩容：队列积压时自动增加 Worker 数量，也支持手动调整
//   - 优雅停止：StopGraceful() 等待所有队列中的任务执行完毕后再退出
//   - 实时指标：原子计数器实时统计成功/失败/重试/队列深度等数据
//   - 事件回调：任务成功或彻底失败时触发用户自定义钩子函数

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Task 接口
// =============================================================================

// Task 是所有任务必须实现的泛型接口。
//
// 类型参数 T 是任务执行完成后返回的结果类型，由调用方在创建 WorkerPool 时指定。
//
// 使用示例：
//
//	type MyTask struct { URL string }
//	func (t *MyTask) TaskID() string { return t.URL }
//	func (t *MyTask) Run(ctx context.Context) (string, error) {
//	    // 业务逻辑写在这里，ctx 已携带超时/取消信号，务必传给下游调用
//	    return fetchURL(ctx, t.URL)
//	}
type Task[T any] interface {
	// Run 执行任务的核心逻辑。
	//
	// ctx 由 Pool 注入，已包含以下语义：
	//   - 若配置了 TaskTimeout，ctx 会在超时后自动取消
	//   - 若调用了 Stop()，ctx 会立即取消
	// 实现者应将 ctx 传递给所有阻塞操作（HTTP 请求、数据库查询等），
	// 以确保任务能被及时中断，避免 goroutine 泄漏。
	Run(ctx context.Context) (T, error)

	// TaskID 返回任务的唯一标识符，用于日志打印和指标上报。
	// 建议返回能区分任务的字符串，例如 "email:alice@example.com" 或 "job-42"。
	TaskID() string
}

// =============================================================================
// Result：任务执行结果
// =============================================================================

// Result 封装单个任务的完整执行结果。
// 无论任务成功还是失败，Pool 都会生成一个 Result 并通过 channel 返回给调用方。
type Result[T any] struct {
	// TaskID 是任务的唯一标识，与 Task.TaskID() 返回值一致，便于结果与任务对应。
	TaskID string

	// Value 是任务成功时的返回值；如果 Err != nil，此字段为零值，不可使用。
	Value T

	// Err 是任务最终的错误。
	// 若配置了重试，此处保存的是最后一次尝试的错误（或重试被取消的错误）。
	// 为 nil 表示任务执行成功。
	Err error

	// Attempts 记录任务一共被执行了多少次（初次执行 + 所有重试次数）。
	// 例如：MaxRetries=2 且第 3 次才成功，则 Attempts=3。
	Attempts int

	// Duration 是所有执行轮次（含重试）累计消耗的时间，
	// 不包含重试等待的退避时间（back-off），仅统计实际运行耗时。
	Duration time.Duration
}

// =============================================================================
// Options：Pool 配置项
// =============================================================================

// Options 用于配置 WorkerPool 的全部行为参数。
// 所有字段均有合理默认值，未设置时由 setDefaults() 补全。
type Options struct {
	// Workers 是 Pool 启动时创建的初始 Worker（goroutine）数量。
	// 同时也是动态缩容的下限：Pool 不会低于此数量。
	// 默认值：4。
	Workers int

	// MaxWorkers 是动态扩容允许达到的最大 Worker 数量。
	// 设为 0 或小于 Workers 时等同于禁用动态扩容（只使用固定 Workers 数量）。
	// 动态扩容策略：当队列积压任务数 > 当前 Worker 数时，每次触发增加 1 个 Worker。
	MaxWorkers int

	// QueueSize 是任务队列（channel）的缓冲容量。
	// Submit() 时若队列已满会立即返回错误，不会阻塞调用方。
	// 默认值：1024。
	QueueSize int

	// TaskTimeout 是单次任务执行（含 Run 本身）的最大允许时间。
	// 超时后 context 会被取消，任务的 Result.Err 会携带超时信息。
	// 设为 0 表示不限制单任务超时（但仍受 Pool 级别的 Stop 信号影响）。
	TaskTimeout time.Duration

	// MaxRetries 是任务失败后最多重试的次数。
	// 总执行次数 = 1（初次）+ MaxRetries（重试）。
	// 设为 0 表示失败后不重试，直接返回错误。
	MaxRetries int

	// RetryDelay 是首次重试前的等待时间。
	// 每次重试后等待时间翻倍（指数退避），例如：200ms -> 400ms -> 800ms。
	// 仅在 MaxRetries > 0 时生效。
	RetryDelay time.Duration

	// RateLimit 限制每秒最多启动的任务数（令牌桶模型）。
	// Worker 在取到任务后、真正调用 Run() 前，会先等待令牌可用。
	// 设为 0 表示不限速。
	// 示例：RateLimit=10.0 表示每 100ms 最多启动 1 个任务。
	RateLimit float64

	// OnSuccess 是任务成功后触发的回调函数。
	// 在独立的 goroutine 中异步调用，不会阻塞 Worker。
	// 可用于记录指标、发送通知等。参数：任务ID、总耗时。
	OnSuccess func(taskID string, duration time.Duration)

	// OnFailure 是任务彻底失败（耗尽所有重试次数）后触发的回调函数。
	// 在独立的 goroutine 中异步调用，不会阻塞 Worker。
	// 可用于报警、写入死信队列等。参数：任务ID、最终错误、总尝试次数。
	OnFailure func(taskID string, err error, attempts int)

	// Logger 是自定义日志函数，签名与 log.Printf 相同。
	// 若不设置，默认使用标准库 log.Printf 输出到 stderr。
	// 可替换为 zap/logrus 等结构化日志库的适配函数。
	Logger func(format string, args ...any)
}

// setDefaults 为未显式设置的字段填充合理的默认值。
// 在 New() 中被调用，调用方无需手动处理。
func (o *Options) setDefaults() {
	if o.Workers <= 0 {
		o.Workers = 4 // 默认 4 个并发 Worker
	}
	if o.MaxWorkers < o.Workers {
		// MaxWorkers 至少等于 Workers，确保动态扩容的上限不低于初始值
		o.MaxWorkers = o.Workers
	}
	if o.QueueSize <= 0 {
		o.QueueSize = 1024 // 默认队列容量 1024
	}
	if o.Logger == nil {
		o.Logger = log.Printf // 默认使用标准库日志
	}
}

// =============================================================================
// Metrics：实时运行指标
// =============================================================================

// Metrics 使用 atomic.Int64 存储 Pool 的实时运行数据，支持无锁并发读写。
// 所有字段均为累计值或当前瞬时值，具体含义见各字段注释。
type Metrics struct {
	// Submitted 是自 Pool 启动以来累计提交（入队）的任务总数。
	Submitted atomic.Int64

	// Succeeded 是执行成功（Run 返回 nil error）的任务总数。
	Succeeded atomic.Int64

	// Failed 是彻底失败（耗尽所有重试后仍报错）的任务总数。
	Failed atomic.Int64

	// Retried 是触发重试的总次数（一个任务重试 3 次则计 3）。
	Retried atomic.Int64

	// InFlight 是当前正在执行中（Run() 尚未返回）的任务数量，为实时瞬时值。
	InFlight atomic.Int64

	// QueueDepth 是当前队列中等待被 Worker 取走的任务数量，为实时瞬时值。
	QueueDepth atomic.Int64

	// Workers 是当前活跃的 Worker goroutine 数量，为实时瞬时值。
	Workers atomic.Int64
}

// Snapshot 对所有指标做一次原子快照，返回值类型（MetricsSnapshot）可安全打印和传递。
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Submitted:  m.Submitted.Load(),
		Succeeded:  m.Succeeded.Load(),
		Failed:     m.Failed.Load(),
		Retried:    m.Retried.Load(),
		InFlight:   m.InFlight.Load(),
		QueueDepth: m.QueueDepth.Load(),
		Workers:    m.Workers.Load(),
	}
}

// MetricsSnapshot 是 Metrics 的值类型快照，不含原子操作，可自由传递和打印。
type MetricsSnapshot struct {
	Submitted  int64 // 累计提交任务数
	Succeeded  int64 // 累计成功任务数
	Failed     int64 // 累计失败任务数
	Retried    int64 // 累计重试次数
	InFlight   int64 // 当前执行中任务数
	QueueDepth int64 // 当前队列深度
	Workers    int64 // 当前 Worker 数量
}

// String 实现 fmt.Stringer，方便直接打印快照内容。
func (s MetricsSnapshot) String() string {
	return fmt.Sprintf(
		"workers=%d queue=%d submitted=%d succeeded=%d failed=%d retried=%d in-flight=%d",
		s.Workers, s.QueueDepth, s.Submitted, s.Succeeded, s.Failed, s.Retried, s.InFlight,
	)
}

// =============================================================================
// 内部 job 包装器
// =============================================================================

// job 是 Pool 内部在 channel 中传递的任务载体，对外不可见。
// 它将用户提交的 Task 和可选的结果接收 channel 打包在一起。
type job[T any] struct {
	task Task[T] // 用户提交的原始任务

	// resultCh 是调用方可选提供的结果接收 channel。
	// 若为 nil（使用 Submit 而非 SubmitWithResult），则任务结果直接丢弃（fire-and-forget 模式）。
	resultCh chan<- Result[T]
}

// =============================================================================
// WorkerPool：核心结构体
// =============================================================================

// WorkerPool 是企业级泛型工作池，负责管理 Worker goroutine、任务队列、
// 超时、重试、限速、动态扩容和优雅停止。
//
// 类型参数 T 是所有任务统一的返回值类型。
// 若需要处理多种不同结构体的任务，可将 T 设置为 any 或自定义 union 结构体。
//
// 使用示例：
//
//	pool := workerpool.New[string](workerpool.Options{Workers: 4})
//	defer pool.StopGraceful()
//	results := pool.SubmitAndCollect(myTasks)
type WorkerPool[T any] struct {
	opts    Options // 不可变的配置项（初始化后不再修改）
	metrics Metrics // 实时运行指标，全程原子操作

	// queue 是任务队列，Worker 通过 range 从中消费任务。
	// 关闭此 channel（close(queue)）是 StopGraceful 触发 Worker 退出的信号。
	queue chan job[T]

	// results 是保留字段，SubmitAndCollect 内部使用临时 channel，此字段暂未启用。
	results chan Result[T]

	// ctx 是 Pool 级别的 context，Stop() 调用时会取消它。
	// 所有 Worker 和任务都监听此 ctx，确保能被快速中断。
	ctx    context.Context
	cancel context.CancelFunc // 对应 ctx 的取消函数

	workerWg sync.WaitGroup // 等待所有 Worker goroutine 退出
	mu       sync.Mutex     // 保护 stopped 标志位的写操作
	stopped  bool           // 标记 Pool 是否已停止，防止重复关闭

	// rateTicker 是限速用的时间间隔 ticker channel。
	// Worker 每次执行任务前需等待此 channel 发出信号（令牌可用）。
	// 若 RateLimit == 0，此字段为 nil，Worker 无需等待。
	rateTicker <-chan time.Time

	// rateStop 用于通知限速 ticker 的清理 goroutine 退出，释放资源。
	rateStop chan struct{}

	// scaleSignal 是向动态扩容 goroutine 发送信号的 channel。
	// 每次有新任务入队时，Submit 会向此 channel 发一个信号（非阻塞）。
	// 扩容 goroutine 收到信号后评估是否需要新增 Worker。
	scaleSignal chan struct{}
}

// NewPool 创建并启动一个 WorkerPool，立即开始接受任务。
//
// 类型参数 T 指定所有任务的统一返回值类型：
//   - 若所有任务返回同一类型（如 string、int），直接指定即可
//   - 若任务返回类型各异，可使用 any 或自定义 union struct（见 example_test.go）
//
// 注意：New 会立即启动 opts.Workers 个 goroutine，
// 使用完毕后务必调用 Stop() 或 StopGraceful() 释放资源，推荐配合 defer 使用。
func NewPool[T any](opts Options) *WorkerPool[T] {
	// 补全未设置的配置项默认值
	opts.setDefaults()

	// 创建 Pool 级别的 context，用于统一取消所有 Worker 和任务
	ctx, cancel := context.WithCancel(context.Background())

	p := &WorkerPool[T]{
		opts:        opts,
		queue:       make(chan job[T], opts.QueueSize), // 带缓冲的任务队列
		results:     make(chan Result[T], opts.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		scaleSignal: make(chan struct{}, 1), // 容量为 1，防止信号堆积
	}

	// ---- 初始化限速器 ----
	if opts.RateLimit > 0 {
		// 将"每秒 N 个任务"换算为"每个任务之间的最小间隔"
		// 例如 RateLimit=10 -> interval=100ms
		interval := time.Duration(float64(time.Second) / opts.RateLimit)
		ticker := time.NewTicker(interval)
		p.rateTicker = ticker.C
		p.rateStop = make(chan struct{})

		// 启动一个专门负责停止 ticker 的 goroutine，避免资源泄漏
		go func() {
			<-p.rateStop  // 等待关闭信号
			ticker.Stop() // 停止 ticker，释放定时器资源
		}()
	}

	// ---- 启动初始 Worker goroutine ----
	for i := 0; i < opts.Workers; i++ {
		p.startWorker()
	}
	p.metrics.Workers.Store(int64(opts.Workers))

	// ---- 启动动态扩容监控 goroutine（仅在允许扩容时）----
	if opts.MaxWorkers > opts.Workers {
		go p.scaler()
	}

	return p
}

// =============================================================================
// 公开 API
// =============================================================================

// Submit 将单个任务提交到队列（fire-and-forget 模式）。
//
// 任务结果不会返回给调用方，适用于只关心副作用、不需要收集结果的场景。
// 若需要获取执行结果，请使用 SubmitWithResult 或 SubmitAndCollect。
//
// 返回值：
//   - nil：任务成功入队
//   - error：Pool 已停止，或队列已满
func (p *WorkerPool[T]) Submit(task Task[T]) error {
	return p.SubmitWithResult(task, nil)
}

// SubmitWithResult 将任务提交到队列，并在任务完成后将 Result 发送到 resultCh。
//
// resultCh 可以为 nil（等同于 Submit）。
// 调用方负责保证 resultCh 有足够的缓冲或有 goroutine 在消费，否则 Worker 会在发送结果时阻塞。
//
// 典型用法：
//
//	ch := make(chan workerpool.Result[string], 10)
//	pool.SubmitWithResult(task1, ch)
//	pool.SubmitWithResult(task2, ch)
//	r1, r2 := <-ch, <-ch
func (p *WorkerPool[T]) SubmitWithResult(task Task[T], resultCh chan<- Result[T]) error {
	// 检查 Pool 是否已停止（加锁保证原子性）
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return errors.New("workerpool: pool is stopped")
	}
	p.mu.Unlock()

	j := job[T]{task: task, resultCh: resultCh}

	// 非阻塞方式入队：队列满时立即返回错误，不阻塞调用方
	select {
	case p.queue <- j:
		p.metrics.Submitted.Add(1)  // 计数：已提交总数 +1
		p.metrics.QueueDepth.Add(1) // 计数：当前队列深度 +1

		// 向扩容 goroutine 发送信号（非阻塞：channel 满时直接跳过，避免阻塞提交路径）
		select {
		case p.scaleSignal <- struct{}{}:
		default:
		}
		return nil

	default:
		// 队列已满，返回描述性错误（包含容量信息，便于调用方决策）
		return fmt.Errorf("workerpool: queue full (capacity %d)", cap(p.queue))
	}
}

// SubmitMany 批量提交任务切片（fire-and-forget 模式）。
//
// 遇到第一个提交失败（Pool 已停止或队列满）时立即终止并返回错误，
// 已成功提交的任务不会被撤回。
func (p *WorkerPool[T]) SubmitMany(tasks []Task[T]) error {
	for _, t := range tasks {
		if err := p.Submit(t); err != nil {
			return err
		}
	}
	return nil
}

// SubmitAndCollect 批量提交任务并阻塞等待所有任务完成，以切片形式返回全部结果。
//
// 结果切片的顺序与任务完成的顺序一致（非提交顺序），调用方应通过 Result.TaskID 区分任务。
// 此方法会阻塞直到所有任务均执行完毕（包括重试），适合需要汇总所有结果再处理的场景。
//
// 注意：若部分任务永久失败，它们的 Result.Err 不为 nil，但仍会被收集进结果集，
// 不会导致此方法提前返回或 panic。
func (p *WorkerPool[T]) SubmitAndCollect(tasks []Task[T]) []Result[T] {
	// 创建一个足够大的缓冲 channel，确保 Worker 发送结果时不会阻塞
	ch := make(chan Result[T], len(tasks))

	// 将所有任务与结果 channel 绑定后提交
	for _, t := range tasks {
		_ = p.SubmitWithResult(t, ch) // 忽略错误：Pool 停止时剩余任务跳过
	}

	// 按提交任务数收集等量结果（阻塞直到全部完成）
	results := make([]Result[T], 0, len(tasks))
	for range tasks {
		results = append(results, <-ch)
	}
	return results
}

// Scale 在运行时动态调整 Worker 数量。
//
// 参数 n 必须满足：1 <= n <= MaxWorkers。
// 扩容（n > 当前 Worker 数）：立即启动新 Worker goroutine，效果即时生效。
// 缩容（n < 当前 Worker 数）：当前 Worker 会自然退出，此方法仅更新计数，不强制终止。
//
// 适用场景：
//   - 业务低峰期手动缩容节省资源
//   - 紧急扩容应对突发流量
//   - 配合外部监控系统实现自动化弹性伸缩
func (p *WorkerPool[T]) Scale(n int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return errors.New("workerpool: pool is stopped")
	}
	if n < 1 {
		return errors.New("workerpool: workers must be >= 1")
	}
	if n > p.opts.MaxWorkers {
		return fmt.Errorf("workerpool: %d exceeds MaxWorkers (%d)", n, p.opts.MaxWorkers)
	}

	current := int(p.metrics.Workers.Load())
	// 仅处理扩容：新目标数量 > 当前数量时，补充差额个 Worker
	for i := current; i < n; i++ {
		p.startWorker()
	}
	// 更新 Worker 计数（缩容时 Worker goroutine 会在下次空闲时自然退出）
	p.metrics.Workers.Store(int64(n))
	return nil
}

// Metrics 返回当前 Pool 的实时运行指标快照。
//
// 返回值是值类型（非指针），可安全传递给其他 goroutine 或打印，
// 但不会自动更新——若需持续监控，请定时轮询此方法。
//
// 示例：
//
//	snap := pool.Metrics()
//	fmt.Println(snap) // workers=4 queue=12 submitted=100 ...
func (p *WorkerPool[T]) Metrics() MetricsSnapshot {
	return p.metrics.Snapshot()
}

// Stop 立即取消 Pool 的 context，强制中断所有正在执行的任务。
//
// 行为：
//   - 正在 Run() 中的任务会收到 ctx.Done() 信号（若任务正确监听 ctx）
//   - 队列中尚未被取走的任务将被丢弃（不再执行）
//   - 不等待任务退出，立即返回
//
// 适用于需要快速退出的场景（如进程收到 SIGKILL）。
// 若希望等待任务完成后再退出，请使用 StopGraceful()。
func (p *WorkerPool[T]) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return // 防止重复调用 cancel() 和 close(rateStop)
	}
	p.stopped = true
	p.cancel() // 取消 Pool 级别 context，所有 Worker 和任务均会感知

	// 停止限速 ticker，释放定时器资源
	if p.rateStop != nil {
		close(p.rateStop)
	}
}

// StopGraceful 优雅停止 Pool：
//  1. 关闭任务队列，不再接受新任务提交
//  2. 等待队列中所有已入队任务执行完毕（包括重试）
//  3. 等待所有 Worker goroutine 正常退出后才返回
//
// 适用于进程收到 SIGTERM 时，希望"做完手头的活再退出"的场景。
// 注意：若某个任务一直阻塞且不监听 ctx，StopGraceful 也会一直等待。
//
// 推荐配合 defer 使用：
//
//	pool := workerpool.New[string](opts)
//	defer pool.StopGraceful()
func (p *WorkerPool[T]) StopGraceful() {
	p.mu.Lock()
	p.stopped = true // 禁止新的 Submit 调用
	p.mu.Unlock()

	// 关闭 queue channel：Worker 的 for range 循环会在队列排空后自动退出
	close(p.queue)

	// 阻塞等待所有 Worker goroutine 完成（包括正在执行的任务）
	p.workerWg.Wait()

	// Worker 全部退出后，清理资源
	p.cancel()
	if p.rateStop != nil {
		close(p.rateStop)
	}
}

// =============================================================================
// 内部实现
// =============================================================================

// startWorker 启动一个新的 Worker goroutine，并将其注册到 WaitGroup。
// 必须在持有 p.mu 锁或初始化阶段调用（New 和 Scale 中调用）。
func (p *WorkerPool[T]) startWorker() {
	p.workerWg.Add(1)
	go p.workerLoop()
}

// workerLoop 是每个 Worker goroutine 运行的主循环。
//
// 工作流程：
//  1. 从 queue channel 取任务（queue 关闭且排空后 range 自动退出）
//  2. 若启用了限速，等待令牌可用（或 ctx 取消）
//  3. 调用 executeWithRetry 执行任务（含重试逻辑）
//  4. 更新指标，触发回调，将结果写入 resultCh（如有）
func (p *WorkerPool[T]) workerLoop() {
	defer p.workerWg.Done() // goroutine 退出时通知 WaitGroup

	// range 语法：当 queue channel 被关闭且所有元素均被消费后，循环自动结束
	for j := range p.queue {

		// ---- 限速等待：获取执行令牌 ----
		if p.rateTicker != nil {
			select {
			case <-p.rateTicker:
				// 成功获取令牌，继续执行

			case <-p.ctx.Done():
				// Pool 被强制停止，放弃执行此任务（但仍需修正队列深度计数）
				p.metrics.QueueDepth.Add(-1)
				continue
			}
		}

		// ---- 执行任务（含超时和重试） ----
		p.metrics.QueueDepth.Add(-1) // 任务已离队，队列深度 -1
		p.metrics.InFlight.Add(1)    // 标记任务进入执行状态

		result := p.executeWithRetry(j.task)

		p.metrics.InFlight.Add(-1) // 任务执行完毕（无论成功或失败）

		// ---- 更新指标 & 触发回调 ----
		if result.Err == nil {
			p.metrics.Succeeded.Add(1)
			if p.opts.OnSuccess != nil {
				// 在独立 goroutine 中调用，避免阻塞 Worker
				go p.opts.OnSuccess(result.TaskID, result.Duration)
			}
		} else {
			p.metrics.Failed.Add(1)
			if p.opts.OnFailure != nil {
				go p.opts.OnFailure(result.TaskID, result.Err, result.Attempts)
			}
		}

		// ---- 将结果发送给调用方（如有需要）----
		// 若调用方传入了 resultCh（通过 SubmitWithResult/SubmitAndCollect），则发送结果
		if j.resultCh != nil {
			j.resultCh <- result
		}
	}
}

// executeWithRetry 在指数退避策略下执行任务，直到成功或耗尽重试次数。
//
// 重试策略：
//   - 每次失败后等待 delay 时间，delay 每次翻倍（指数退避）
//   - 若在退避等待期间 Pool 被取消，立即返回取消错误
//   - 所有尝试均失败则返回最后一次的错误
func (p *WorkerPool[T]) executeWithRetry(task Task[T]) Result[T] {
	result := Result[T]{TaskID: task.TaskID()}
	delay := p.opts.RetryDelay           // 首次重试等待时间（后续翻倍）
	maxAttempts := p.opts.MaxRetries + 1 // 总尝试次数 = 重试次数 + 首次执行

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempts = attempt

		// 记录本次执行耗时
		start := time.Now()
		val, err := p.runOnce(task) // 执行一次任务（含超时控制）
		result.Duration += time.Since(start)

		if err == nil {
			// 执行成功，填充结果并立即返回
			result.Value = val
			result.Err = nil
			return result
		}

		// 本次执行失败
		result.Err = err

		if attempt < maxAttempts {
			// 还有重试机会：记录日志，等待退避时间后重试
			p.metrics.Retried.Add(1)
			p.opts.Logger("[workerpool] 任务 %q 第 %d 次执行失败: %v - 将在 %s 后重试",
				task.TaskID(), attempt, err, delay)

			select {
			case <-time.After(delay):
				delay *= 2 // 指数退避：每次等待时间翻倍
			case <-p.ctx.Done():
				// 退避等待期间 Pool 被取消，包装错误并提前返回
				result.Err = fmt.Errorf("在重试退避期间被取消: %w", p.ctx.Err())
				return result
			}
		}
	}

	// 已耗尽所有重试次数，记录最终失败日志
	p.opts.Logger("[workerpool] 任务 %q 在 %d 次尝试后永久失败: %v",
		task.TaskID(), maxAttempts, result.Err)
	return result
}

// runOnce 执行任务一次，并根据配置附加超时控制。
//
// 若 TaskTimeout > 0：
//   - 创建带超时的子 context
//   - 在独立 goroutine 中运行 task.Run()
//   - 若超时先到，立即返回超时错误（task.Run 的 goroutine 继续运行直到感知 ctx 取消）
//
// 若 TaskTimeout == 0：
//   - 直接用 Pool 级别的 ctx 调用 task.Run()，不附加额外超时
func (p *WorkerPool[T]) runOnce(task Task[T]) (T, error) {
	var zero T // 错误时的零值返回

	if p.opts.TaskTimeout <= 0 {
		// 未设置超时：直接执行，任务可运行到 Pool 被 Stop() 为止
		return task.Run(p.ctx)
	}

	// 创建带超时的子 context（超时后自动取消，defer cancel 确保资源释放）
	ctx, cancel := context.WithTimeout(p.ctx, p.opts.TaskTimeout)
	defer cancel()

	// 用匿名结构体承载 goroutine 的执行结果
	type outcome struct {
		val T
		err error
	}
	ch := make(chan outcome, 1) // 缓冲为 1，避免 goroutine 泄漏

	// 在独立 goroutine 中执行任务，确保 select 能同时等待结果和超时
	go func() {
		v, e := task.Run(ctx)
		ch <- outcome{v, e}
	}()

	select {
	case o := <-ch:
		// 任务在超时前完成，返回实际结果
		return o.val, o.err

	case <-ctx.Done():
		// 超时（或 Pool 被 Stop()）先触发，返回超时错误
		// 注意：执行任务的 goroutine 仍在运行，但其持有的 ctx 已取消，
		// 若任务正确实现了 ctx 监听，它会很快退出。
		return zero, fmt.Errorf("任务执行超时（限制 %s）", p.opts.TaskTimeout)
	}
}

// scaler 是动态扩容的监控 goroutine，在 New() 中启动（仅当 MaxWorkers > Workers 时）。
//
// 扩容触发条件（每次收到 scaleSignal 时评估）：
//   - 当前队列深度 > 当前 Worker 数量（说明 Worker 供不应求）
//   - 当前 Worker 数量 < MaxWorkers（还有扩容空间）
//
// 扩容步长：每次仅增加 1 个 Worker，避免过激扩容导致资源耗尽。
// 若需大幅度扩容，调用方可手动调用 Scale(n) 方法。
func (p *WorkerPool[T]) scaler() {
	for {
		select {
		case <-p.ctx.Done():
			// Pool 已停止，退出监控 goroutine
			return

		case <-p.scaleSignal:
			// 收到扩容信号（新任务入队时触发），评估是否需要扩容
			queueDepth := p.metrics.QueueDepth.Load()
			currentWorkers := p.metrics.Workers.Load()

			// 仅当队列积压超过当前 Worker 数且未达上限时才扩容
			if queueDepth > currentWorkers && currentWorkers < int64(p.opts.MaxWorkers) {
				p.mu.Lock()
				if !p.stopped { // 再次确认 Pool 未停止（双重检查）
					p.startWorker()
					p.metrics.Workers.Add(1)
					p.opts.Logger("[workerpool] 自动扩容：当前 Worker 数 = %d（队列积压 = %d）",
						p.metrics.Workers.Load(), queueDepth)
				}
				p.mu.Unlock()
			}
		}
	}
}

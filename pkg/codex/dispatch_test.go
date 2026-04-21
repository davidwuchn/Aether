package codex

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// --- GroupByWave tests ---

func TestGroupByWave_SingleWave(t *testing.T) {
	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Wave: 1},
		{WorkerName: "Hammer-2", Wave: 1},
	}

	groups := GroupByWave(dispatches)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[1]) != 2 {
		t.Fatalf("expected 2 dispatches in wave 1, got %d", len(groups[1]))
	}
	if groups[1][0].WorkerName != "Hammer-1" {
		t.Errorf("first dispatch = %q, want %q", groups[1][0].WorkerName, "Hammer-1")
	}
	if groups[1][1].WorkerName != "Hammer-2" {
		t.Errorf("second dispatch = %q, want %q", groups[1][1].WorkerName, "Hammer-2")
	}
}

func TestGroupByWave_MultipleWaves(t *testing.T) {
	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Wave: 1},
		{WorkerName: "Hammer-2", Wave: 2},
		{WorkerName: "Hammer-3", Wave: 1},
		{WorkerName: "Hammer-4", Wave: 3},
	}

	groups := GroupByWave(dispatches)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups[1]) != 2 {
		t.Errorf("wave 1: expected 2 dispatches, got %d", len(groups[1]))
	}
	if len(groups[2]) != 1 {
		t.Errorf("wave 2: expected 1 dispatch, got %d", len(groups[2]))
	}
	if len(groups[3]) != 1 {
		t.Errorf("wave 3: expected 1 dispatch, got %d", len(groups[3]))
	}
}

func TestGroupByWave_EmptyInput(t *testing.T) {
	groups := GroupByWave(nil)
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups for nil input, got %d", len(groups))
	}

	groups = GroupByWave([]WorkerDispatch{})
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups for empty input, got %d", len(groups))
	}
}

// --- DispatchBatch tests ---

func TestDispatchBatch_SequentialExecution(t *testing.T) {
	invoker := &countingInvoker{}
	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1, SkillSection: "TDD discipline", PheromoneSection: "FOCUS: security"},
		{WorkerName: "Hammer-2", Caste: "builder", TaskID: "1.2", Wave: 1},
		{WorkerName: "Ranger-1", Caste: "scout", TaskID: "2.1", Wave: 2},
	}

	results, err := DispatchBatch(context.Background(), invoker, dispatches)
	if err != nil {
		t.Fatalf("DispatchBatch returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// All should be completed
	for i, r := range results {
		if r.Status != "completed" {
			t.Errorf("result[%d].Status = %q, want %q", i, r.Status, "completed")
		}
	}

	// Verify execution order: wave 1 before wave 2
	if invoker.callOrder[0] != "Hammer-1" {
		t.Errorf("first call = %q, want %q", invoker.callOrder[0], "Hammer-1")
	}
	if invoker.callOrder[1] != "Hammer-2" {
		t.Errorf("second call = %q, want %q", invoker.callOrder[1], "Hammer-2")
	}
	if invoker.callOrder[2] != "Ranger-1" {
		t.Errorf("third call = %q, want %q", invoker.callOrder[2], "Ranger-1")
	}

	if invoker.totalCalls != 3 {
		t.Errorf("total invoker calls = %d, want %d", invoker.totalCalls, 3)
	}
}

func TestDispatchBatch_FailedWorkerDoesNotBlockNextWave(t *testing.T) {
	invoker := &failingInvoker{failNames: map[string]bool{"Hammer-1": true}}

	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1},
		{WorkerName: "Hammer-2", Caste: "builder", TaskID: "1.2", Wave: 2},
	}

	results, err := DispatchBatch(context.Background(), invoker, dispatches)
	if err != nil {
		t.Fatalf("DispatchBatch should not return error even with failures, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Hammer-1 should be failed
	if results[0].Status != "failed" {
		t.Errorf("result[0].Status = %q, want %q", results[0].Status, "failed")
	}
	if results[0].Error == nil {
		t.Error("result[0].Error should not be nil")
	}

	// Hammer-2 should still have been executed (next wave not blocked)
	if results[1].Status != "completed" {
		t.Errorf("result[1].Status = %q, want %q (wave 2 should still execute)", results[1].Status, "completed")
	}
}

func TestDispatchBatch_AllWorkersFail(t *testing.T) {
	invoker := &alwaysFailInvoker{}

	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1},
		{WorkerName: "Hammer-2", Caste: "builder", TaskID: "1.2", Wave: 1},
	}

	results, err := DispatchBatch(context.Background(), invoker, dispatches)
	if err != nil {
		t.Fatalf("DispatchBatch should not return error, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if r.Status != "failed" {
			t.Errorf("result[%d].Status = %q, want %q", i, r.Status, "failed")
		}
	}
}

func TestDispatchBatch_Timeout(t *testing.T) {
	invoker := &slowInvoker{delay: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1},
	}

	results, err := DispatchBatch(ctx, invoker, dispatches)
	// Should still return results (not an error), with the worker marked as failed/timeout
	if err != nil {
		t.Fatalf("DispatchBatch returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "timeout" {
		t.Errorf("result[0].Status = %q, want %q", results[0].Status, "timeout")
	}
}

func TestDispatchBatch_PropagatesSkillAndPheromoneSections(t *testing.T) {
	invoker := &capturingInvoker{}
	dispatches := []WorkerDispatch{
		{
			WorkerName:       "Hammer-1",
			Caste:            "builder",
			TaskID:           "1.1",
			Wave:             1,
			SkillSection:     "TDD discipline",
			PheromoneSection: "FOCUS: security\nREDIRECT: no globals",
		},
	}

	results, err := DispatchBatch(context.Background(), invoker, dispatches)
	if err != nil {
		t.Fatalf("DispatchBatch returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "completed" {
		t.Fatalf("result status = %q, want completed", results[0].Status)
	}

	if invoker.lastConfig.SkillSection != "TDD discipline" {
		t.Errorf("SkillSection = %q, want %q", invoker.lastConfig.SkillSection, "TDD discipline")
	}
	if invoker.lastConfig.PheromoneSection != "FOCUS: security\nREDIRECT: no globals" {
		t.Errorf("PheromoneSection = %q, want %q", invoker.lastConfig.PheromoneSection, "FOCUS: security\nREDIRECT: no globals")
	}
}

func TestDispatchBatchWithObserver_EmitsLifecycleTransitions(t *testing.T) {
	invoker := &countingInvoker{}
	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1},
	}

	var statuses []string
	var workerNames []string
	observer := func(event DispatchLifecycleEvent) {
		statuses = append(statuses, event.Status)
		workerNames = append(workerNames, event.Dispatch.WorkerName)
	}

	results, err := DispatchBatchWithObserver(context.Background(), invoker, dispatches, observer)
	if err != nil {
		t.Fatalf("DispatchBatchWithObserver returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	wantStatuses := []string{"starting", "running", "completed"}
	if len(statuses) != len(wantStatuses) {
		t.Fatalf("observer event count = %d, want %d (%v)", len(statuses), len(wantStatuses), statuses)
	}
	for i, want := range wantStatuses {
		if statuses[i] != want {
			t.Errorf("statuses[%d] = %q, want %q", i, statuses[i], want)
		}
		if workerNames[i] != "Hammer-1" {
			t.Errorf("workerNames[%d] = %q, want %q", i, workerNames[i], "Hammer-1")
		}
	}
}

func TestDispatchBatchWithObserver_DoesNotInventRunningWithoutProgressSupport(t *testing.T) {
	invoker := &capturingInvoker{}
	dispatches := []WorkerDispatch{
		{WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1", Wave: 1},
	}

	var statuses []string
	observer := func(event DispatchLifecycleEvent) {
		statuses = append(statuses, event.Status)
	}

	results, err := DispatchBatchWithObserver(context.Background(), invoker, dispatches, observer)
	if err != nil {
		t.Fatalf("DispatchBatchWithObserver returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	wantStatuses := []string{"starting", "completed"}
	if len(statuses) != len(wantStatuses) {
		t.Fatalf("observer event count = %d, want %d (%v)", len(statuses), len(wantStatuses), statuses)
	}
	for i, want := range wantStatuses {
		if statuses[i] != want {
			t.Errorf("statuses[%d] = %q, want %q", i, statuses[i], want)
		}
	}
}

// --- ExtractClaims tests ---

func TestExtractClaims_AggregatesResults(t *testing.T) {
	results := []DispatchResult{
		{
			WorkerName: "Hammer-1",
			Status:     "completed",
			WorkerResult: &WorkerResult{
				FilesCreated:  []string{"/tmp/a.go", "/tmp/b.go"},
				FilesModified: []string{"/tmp/c.go"},
				TestsWritten:  []string{"/tmp/a_test.go"},
			},
		},
		{
			WorkerName: "Hammer-2",
			Status:     "completed",
			WorkerResult: &WorkerResult{
				FilesCreated:  []string{"/tmp/d.go"},
				FilesModified: []string{"/tmp/e.go", "/tmp/f.go"},
				TestsWritten:  []string{"/tmp/d_test.go", "/tmp/e_test.go"},
			},
		},
		{
			WorkerName: "Hammer-3",
			Status:     "failed",
			WorkerResult: &WorkerResult{
				FilesCreated:  []string{"/tmp/should_not_include.go"},
				FilesModified: []string{"/tmp/should_not_include2.go"},
				TestsWritten:  []string{"/tmp/should_not_include_test.go"},
			},
		},
	}

	claims := ExtractClaims(results)

	if len(claims.FilesCreated) != 3 {
		t.Errorf("FilesCreated count = %d, want 3 (failed worker excluded)", len(claims.FilesCreated))
	}
	if len(claims.FilesModified) != 3 {
		t.Errorf("FilesModified count = %d, want 3 (failed worker excluded)", len(claims.FilesModified))
	}
	if len(claims.TestsWritten) != 3 {
		t.Errorf("TestsWritten count = %d, want 3 (failed worker excluded)", len(claims.TestsWritten))
	}

	// Verify the failed worker's files are NOT included
	for _, f := range claims.FilesCreated {
		if f == "/tmp/should_not_include.go" {
			t.Error("failed worker's FilesCreated should not be included")
		}
	}
}

func TestExtractClaims_NoSuccessfulResults(t *testing.T) {
	results := []DispatchResult{
		{
			WorkerName: "Hammer-1",
			Status:     "failed",
			WorkerResult: &WorkerResult{
				FilesCreated:  []string{"/tmp/a.go"},
				FilesModified: []string{"/tmp/b.go"},
				TestsWritten:  []string{"/tmp/a_test.go"},
			},
		},
	}

	claims := ExtractClaims(results)

	if len(claims.FilesCreated) != 0 {
		t.Errorf("FilesCreated should be empty, got %d items", len(claims.FilesCreated))
	}
	if len(claims.FilesModified) != 0 {
		t.Errorf("FilesModified should be empty, got %d items", len(claims.FilesModified))
	}
	if len(claims.TestsWritten) != 0 {
		t.Errorf("TestsWritten should be empty, got %d items", len(claims.TestsWritten))
	}
}

// --- Test helpers ---

// capturingInvoker captures the last WorkerConfig it received.
type capturingInvoker struct {
	lastConfig WorkerConfig
}

func (c *capturingInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	c.lastConfig = config
	return WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    fmt.Sprintf("%s completed %s", config.WorkerName, config.TaskID),
		Duration:   time.Millisecond,
	}, nil
}

func (c *capturingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (c *capturingInvoker) ValidateAgent(path string) error      { return nil }

// countingInvoker records the order of calls.
type countingInvoker struct {
	callOrder  []string
	totalCalls int
}

func (c *countingInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return c.InvokeWithProgress(ctx, config, nil)
}

func (c *countingInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	c.callOrder = append(c.callOrder, config.WorkerName)
	c.totalCalls++
	emitWorkerProgress(observer, WorkerProgressEvent{
		Status:     "running",
		Message:    "counting invoker heartbeat observed",
		OccurredAt: time.Now().UTC(),
	})
	return WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    fmt.Sprintf("%s completed %s", config.WorkerName, config.TaskID),
		Duration:   time.Millisecond,
	}, nil
}

func (c *countingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (c *countingInvoker) ValidateAgent(path string) error      { return nil }

// failingInvoker fails for specific worker names.
type failingInvoker struct {
	FakeInvoker
	failNames map[string]bool
}

func (f *failingInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	if f.failNames[config.WorkerName] {
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "failed",
			Error:      errors.New("simulated failure"),
			Duration:   time.Millisecond,
		}, nil
	}
	return f.FakeInvoker.Invoke(ctx, config)
}

func (f *failingInvoker) InvokeWithProgress(ctx context.Context, config WorkerConfig, observer WorkerProgressObserver) (WorkerResult, error) {
	if f.failNames[config.WorkerName] {
		return f.Invoke(ctx, config)
	}
	return f.FakeInvoker.InvokeWithProgress(ctx, config, observer)
}

// alwaysFailInvoker always returns a failed result.
type alwaysFailInvoker struct{}

func (a *alwaysFailInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	return WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "failed",
		Error:      errors.New("always fails"),
		Duration:   time.Millisecond,
	}, nil
}

func (a *alwaysFailInvoker) IsAvailable(ctx context.Context) bool { return true }
func (a *alwaysFailInvoker) ValidateAgent(path string) error      { return nil }

// slowInvoker simulates a worker that takes a long time.
type slowInvoker struct {
	delay time.Duration
}

func (s *slowInvoker) Invoke(ctx context.Context, config WorkerConfig) (WorkerResult, error) {
	select {
	case <-time.After(s.delay):
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "completed",
			Duration:   s.delay,
		}, nil
	case <-ctx.Done():
		return WorkerResult{
			WorkerName: config.WorkerName,
			Caste:      config.Caste,
			TaskID:     config.TaskID,
			Status:     "timeout",
			Error:      ctx.Err(),
			Duration:   s.delay,
		}, nil
	}
}

func (s *slowInvoker) IsAvailable(ctx context.Context) bool { return true }
func (s *slowInvoker) ValidateAgent(path string) error      { return nil }

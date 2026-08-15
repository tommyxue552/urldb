package task

import (
	"time"

	"github.com/ctwj/urldb/db/entity"
)

// TaskOperationsSnapshot is a low-sensitivity operational view used to decide
// whether the process-local executor needs a durable queue/worker migration.
// It intentionally contains aggregate counts and timings only.
type TaskOperationsSnapshot struct {
	MeasuredAt                      time.Time        `json:"measured_at"`
	WindowStart                     time.Time        `json:"window_start"`
	WindowDays                      int              `json:"window_days"`
	TasksCreated                    int64            `json:"tasks_created"`
	TasksByType                     map[string]int64 `json:"tasks_by_type"`
	TasksByStatus                   map[string]int64 `json:"tasks_by_status"`
	TaskItemsByStatus               map[string]int64 `json:"task_items_by_status"`
	PendingTasks                    int64            `json:"pending_tasks"`
	RunningTasks                    int64            `json:"running_tasks"`
	CurrentProcessRunningTasks      int              `json:"current_process_running_tasks"`
	PendingItems                    int64            `json:"pending_items"`
	ProcessingItems                 int64            `json:"processing_items"`
	OldestPendingTaskAgeSeconds     float64          `json:"oldest_pending_task_age_seconds"`
	OldestPendingItemAgeSeconds     float64          `json:"oldest_pending_item_age_seconds"`
	CompletedTasksWithTiming        int64            `json:"completed_tasks_with_timing"`
	AverageTaskDurationSeconds      float64          `json:"average_task_duration_seconds"`
	MaximumTaskDurationSeconds      float64          `json:"maximum_task_duration_seconds"`
	RecoveredTasksSinceProcessStart int64            `json:"recovered_tasks_since_process_start"`
}

// GetOperationsSnapshot reads the durable task state and combines it with the
// process-local running/recovery state. The default window is seven days,
// matching the migration baseline documented in docs/task-operations.md.
func (tm *TaskManager) GetOperationsSnapshot(windowDays int) (*TaskOperationsSnapshot, error) {
	if windowDays < 1 || windowDays > 90 {
		windowDays = 7
	}

	now := time.Now()
	windowStart := now.Add(-time.Duration(windowDays) * 24 * time.Hour)
	snapshot := &TaskOperationsSnapshot{
		MeasuredAt:        now,
		WindowStart:       windowStart,
		WindowDays:        windowDays,
		TasksByType:       make(map[string]int64),
		TasksByStatus:     make(map[string]int64),
		TaskItemsByStatus: make(map[string]int64),
	}

	tm.mu.RLock()
	snapshot.CurrentProcessRunningTasks = len(tm.running)
	snapshot.RecoveredTasksSinceProcessStart = tm.recoveredTasks
	tm.mu.RUnlock()

	db := tm.repoMgr.TaskRepository.GetDB()
	var tasks []struct {
		Type        entity.TaskType
		Status      entity.TaskStatus
		StartedAt   *time.Time
		CompletedAt *time.Time
	}
	if err := db.Model(&entity.Task{}).
		Select("type, status, started_at, completed_at").
		Where("created_at >= ?", windowStart).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	snapshot.TasksCreated = int64(len(tasks))
	for _, current := range tasks {
		snapshot.TasksByType[string(current.Type)]++
		snapshot.TasksByStatus[string(current.Status)]++
		if current.CompletedAt == nil || current.StartedAt == nil {
			continue
		}
		duration := current.CompletedAt.Sub(*current.StartedAt).Seconds()
		if duration < 0 {
			continue
		}
		snapshot.CompletedTasksWithTiming++
		snapshot.AverageTaskDurationSeconds += duration
		if duration > snapshot.MaximumTaskDurationSeconds {
			snapshot.MaximumTaskDurationSeconds = duration
		}
	}
	if snapshot.CompletedTasksWithTiming > 0 {
		snapshot.AverageTaskDurationSeconds /= float64(snapshot.CompletedTasksWithTiming)
	}

	var allTaskStatus []struct {
		Status string
		Count  int64
	}
	if err := db.Model(&entity.Task{}).
		Select("status, COUNT(*) AS count").
		Group("status").
		Scan(&allTaskStatus).Error; err != nil {
		return nil, err
	}
	for _, row := range allTaskStatus {
		switch row.Status {
		case string(entity.TaskStatusPending):
			snapshot.PendingTasks = row.Count
		case string(entity.TaskStatusRunning):
			snapshot.RunningTasks = row.Count
		}
	}

	var itemStatus []struct {
		Status string
		Count  int64
	}
	if err := db.Model(&entity.TaskItem{}).
		Select("status, COUNT(*) AS count").
		Group("status").
		Scan(&itemStatus).Error; err != nil {
		return nil, err
	}
	for _, row := range itemStatus {
		snapshot.TaskItemsByStatus[row.Status] = row.Count
		switch row.Status {
		case string(entity.TaskItemStatusPending):
			snapshot.PendingItems = row.Count
		case string(entity.TaskItemStatusProcessing):
			snapshot.ProcessingItems = row.Count
		}
	}

	var oldestTask struct {
		CreatedAt time.Time
	}
	if err := db.Model(&entity.Task{}).
		Select("created_at").
		Where("status = ?", entity.TaskStatusPending).
		Order("created_at ASC").
		First(&oldestTask).Error; err == nil {
		snapshot.OldestPendingTaskAgeSeconds = nonNegativeAgeSeconds(now, oldestTask.CreatedAt)
	}
	var oldestItem struct {
		CreatedAt time.Time
	}
	if err := db.Model(&entity.TaskItem{}).
		Select("created_at").
		Where("status = ?", entity.TaskItemStatusPending).
		Order("created_at ASC").
		First(&oldestItem).Error; err == nil {
		snapshot.OldestPendingItemAgeSeconds = nonNegativeAgeSeconds(now, oldestItem.CreatedAt)
	}

	return snapshot, nil
}

func nonNegativeAgeSeconds(now, createdAt time.Time) float64 {
	age := now.Sub(createdAt).Seconds()
	if age < 0 {
		return 0
	}
	return age
}

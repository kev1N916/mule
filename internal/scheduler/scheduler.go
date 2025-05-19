package scheduler

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
)

// Understood 

type Task struct {
	ID       cron.EntryID
	Schedule string
	Action   func()
}

type Scheduler struct {
	cron   *cron.Cron
	tasks  map[string]*Task
	mu     sync.RWMutex
	Logger logr.Logger
}

func NewScheduler(l logr.Logger) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		tasks:  make(map[string]*Task),
		Logger: l,
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) AddTask(key string, schedule string, action func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing task if it exists
	if existingTask, exists := s.tasks[key]; exists {
		s.cron.Remove(existingTask.ID)
		delete(s.tasks, key)
	}

	id, err := s.cron.AddFunc(schedule, func() {
		s.Logger.Info("Running scheduled task", "key", key)
		action()
	})

	if err != nil {
		return err
	}

	s.tasks[key] = &Task{
		ID:       id,
		Schedule: schedule,
		Action:   action,
	}

	return nil
}

func (s *Scheduler) RemoveTask(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, exists := s.tasks[key]; exists {
		s.cron.Remove(task.ID)
		delete(s.tasks, key)
	}
}

func (s *Scheduler) UpdateTask(key string, schedule string, action func()) error {
	return s.AddTask(key, schedule, action)
}

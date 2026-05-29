package usecase

import "context"

type Job interface {
	Name() string
	Run(ctx context.Context) error
}

type ScheduleSpec struct {
	Expression string
	Timezone   string
}

type Scheduler interface {
	Register(spec ScheduleSpec, job Job) error
	Start() error
	Stop(ctx context.Context) error
}

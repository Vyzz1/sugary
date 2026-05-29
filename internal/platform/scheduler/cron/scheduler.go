package cron

import (
	"context"
	"fmt"
	"strings"

	robfigcron "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"sugary/internal/usecase"
)

type Scheduler struct {
	engine *robfigcron.Cron
}

func New() *Scheduler {
	return &Scheduler{
		engine: robfigcron.New(),
	}
}

func (s *Scheduler) Register(spec usecase.ScheduleSpec, job usecase.Job) error {
	expression := strings.TrimSpace(spec.Expression)
	if expression == "" {
		return fmt.Errorf("schedule expression is required")
	}

	if timezone := strings.TrimSpace(spec.Timezone); timezone != "" {
		expression = "CRON_TZ=" + timezone + " " + expression
	}

	_, err := s.engine.AddFunc(expression, func() {
		zap.L().Info("scheduled_job_started", zap.String("job_name", job.Name()))
		if err := job.Run(context.Background()); err != nil {
			zap.L().Error("scheduled_job_failed",
				zap.String("job_name", job.Name()),
				zap.Error(err),
			)
			return
		}
		zap.L().Info("scheduled_job_finished", zap.String("job_name", job.Name()))
	})
	if err != nil {
		return err
	}

	zap.L().Info("scheduled_job_registered",
		zap.String("job_name", job.Name()),
		zap.String("expression", spec.Expression),
		zap.String("timezone", spec.Timezone),
	)
	return nil
}

func (s *Scheduler) Start() error {
	s.engine.Start()
	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	stopCtx := s.engine.Stop()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

package usecase

type HealthCheck struct{}

func NewHealthCheck() HealthCheck {
	return HealthCheck{}
}

func (HealthCheck) Execute() map[string]string {
	return map[string]string{
		"status": "ok",
	}
}

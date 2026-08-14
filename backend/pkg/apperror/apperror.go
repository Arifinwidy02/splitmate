package apperror

type Validation struct {
	Message string
}

func (e *Validation) Error() string { return e.Message }

func NewValidation(message string) *Validation {
	return &Validation{Message: message}
}

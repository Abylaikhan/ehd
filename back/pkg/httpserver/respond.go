package httpserver

// Единый контракт ошибки из ТЗ:
// {"request_id": "...", "error": {"code": "...", "message": "...", "details": [...]}}

type ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// APIError — доменная HTTP-ошибка, которую возвращают хендлеры;
// центральный ErrorHandler рендерит её в единый контракт.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details []ErrorDetail
}

func (e *APIError) Error() string { return e.Message }

func NewError(status int, code, message string, details ...ErrorDetail) *APIError {
	return &APIError{Status: status, Code: code, Message: message, Details: details}
}

type errorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type errorResponse struct {
	RequestID string    `json:"request_id"`
	Error     errorBody `json:"error"`
}

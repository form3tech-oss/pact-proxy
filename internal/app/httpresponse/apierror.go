package httpresponse

type APIError struct {
	ErrorMessage string `json:"error_message,omitempty"`
}

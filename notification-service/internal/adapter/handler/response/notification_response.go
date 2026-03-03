package response

type ListResponse struct {
	ID      int64  `json:"id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	SendAt  string `json:"send_at"`
}

type DetailResponse struct {
	ID               int64  `json:"id"`
	Subject          string `json:"subject"`
	Message          string `json:"message"`
	Status           string `json:"status"`
	SendAt           string `json:"send_at"`
	ReadAt           string `json:"read_at"`
	NotificationType string `json:"notification_type"`
}

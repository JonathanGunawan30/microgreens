package request

type NotificationReadRequest struct {
	ID int64 `json:"id" validate:"required"`
}

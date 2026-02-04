package entity

type ActionType string

const (
	ActionInsert ActionType = "INSERT"
	ActionDelete ActionType = "DELETE"
)

type EsSyncMessage struct {
	Action ActionType     `json:"action"`
	Data   *ProductEntity `json:"data,omitempty"`
	ID     int64          `json:"id,omitempty"`
}

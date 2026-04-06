package constant

const (
	NOTIF_EMAIL_VERIFICATION        = "email_verification"
	NOTIF_EMAIL_FORGOT_PASSWORD     = "forgot_password"
	NOTIF_EMAIL_UPDATE_STATUS_ORDER = "email-update-status-order"
	NOTIF_EMAIL_CREATE_CUSTOMER     = "create_customer"
	NOTIF_EMAIL_UPDATE_CUSTOMER     = "update_costomer"
	ORDER_PUSH_QUEUE                = "order.created.push.notification"
	ORDER_EMAIL_QUEUE               = "order.created.email.notification"
)

const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
	StatusFailed  = "FAILED"
	TypeEmail     = "EMAIL"
	TypePush      = "PUSH"
)

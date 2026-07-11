package model

type AppWarrantyProfileQueryReq struct {
	AppID *string `form:"appid" validate:"omitempty,max=100"`
}

type AppWarrantyProfileSaveReq struct {
	ContactName  *string `json:"contact_name" validate:"omitempty,max=100"`
	ContactPhone *string `json:"contact_phone" validate:"omitempty,max=50"`
}

type AppWarrantyBatteryCardResp struct {
	DeviceID           string  `json:"device_id"`
	DeviceNumber       string  `json:"device_number"`
	BatterySerial      string  `json:"battery_serial"`
	BatteryModelName   *string `json:"battery_model_name"`
	ActivationDate     *string `json:"activation_date"`
	WarrantyStartDate  *string `json:"warranty_start_date"`
	WarrantyExpireDate *string `json:"warranty_expire_date"`
	WarrantyMonths     *int32  `json:"warranty_months"`
}

type AppWarrantyProfileResp struct {
	ContactName                   *string                      `json:"contact_name"`
	ContactPhone                  *string                      `json:"contact_phone"`
	WarrantyCardsEnabled          bool                         `json:"warranty_cards_enabled"`
	WarrantyProfileExists         bool                         `json:"warranty_profile_exists"`
	WarrantyProfileCompleted      bool                         `json:"warranty_profile_completed"`
	WarrantyProfileReminderNeeded bool                         `json:"warranty_profile_reminder_needed"`
	Batteries                     []AppWarrantyBatteryCardResp `json:"batteries"`
}

type BatteryWarrantyUpdateReq struct {
	WarrantyMonths     *int32  `json:"warranty_months" validate:"omitempty,gte=0,lte=600"`
	WarrantyStartDate  *string `json:"warranty_start_date" validate:"omitempty"`
	WarrantyExpireDate *string `json:"warranty_expire_date" validate:"omitempty"`
}

type BatteryWarrantyUserResp struct {
	UserID       string  `json:"user_id"`
	UserName     *string `json:"user_name"`
	Username     *string `json:"username"`
	UserPhone    string  `json:"user_phone"`
	ContactName  *string `json:"contact_name"`
	ContactPhone *string `json:"contact_phone"`
	IsOwner      bool    `json:"is_owner"`
	BindingTime  *string `json:"binding_time"`
}

type BatteryWarrantyResp struct {
	DeviceID               string                    `json:"device_id"`
	DeviceNumber           string                    `json:"device_number"`
	BatterySerial          string                    `json:"battery_serial"`
	BatteryModelID         *string                   `json:"battery_model_id"`
	BatteryModelName       *string                   `json:"battery_model_name"`
	ActivationDate         *string                   `json:"activation_date"`
	WarrantyStartDate      *string                   `json:"warranty_start_date"`
	WarrantyExpireDate     *string                   `json:"warranty_expire_date"`
	WarrantyMonths         *int32                    `json:"warranty_months"`
	WarrantyManualOverride bool                      `json:"warranty_manual_override"`
	WarrantyUpdatedAt      *string                   `json:"warranty_updated_at"`
	WarrantyUpdatedBy      *string                   `json:"warranty_updated_by"`
	Users                  []BatteryWarrantyUserResp `json:"users"`
}

type BatteryWarrantyRecalcJobCreateResp struct {
	JobID string `json:"job_id"`
}

type BatteryWarrantyRecalcJobStatusResp struct {
	JobID         string  `json:"job_id"`
	Source        string  `json:"source"`
	ScopeModelID  *string `json:"scope_model_id"`
	Status        string  `json:"status"`
	TotalRows     int     `json:"total_rows"`
	ProcessedRows int     `json:"processed_rows"`
	SuccessRows   int     `json:"success_rows"`
	SkippedRows   int     `json:"skipped_rows"`
	FailedRows    int     `json:"failed_rows"`
	ErrorMessage  *string `json:"error_message"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	CreatedAt     string  `json:"created_at"`
}

type BatteryWarrantyRecalcJobLogItem struct {
	ID             int64   `json:"id"`
	Level          string  `json:"level"`
	DeviceID       *string `json:"device_id"`
	DeviceNumber   *string `json:"device_number"`
	BatteryModelID *string `json:"battery_model_id"`
	Message        string  `json:"message"`
	CreatedAt      string  `json:"created_at"`
}

type BatteryWarrantyRecalcJobLogListResp struct {
	List        []BatteryWarrantyRecalcJobLogItem `json:"list"`
	NextAfterID int64                             `json:"next_after_id"`
}

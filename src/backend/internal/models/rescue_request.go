package models

// RescueRequestMaster คือข้อมูลหลักของคำร้องจาก RescueRequest Service.
// Location fields ถูก flatten อยู่ใน master โดยตรง (ตาม new API contract).
type RescueRequestMaster struct {
	RequestID       string  `json:"requestId"`
	IncidentID      string  `json:"incidentId"`
	RequestType     string  `json:"requestType"`
	Description     string  `json:"description,omitempty"`
	PeopleCount     int     `json:"peopleCount"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	LocationDetails string  `json:"locationDetails,omitempty"`
	Province        string  `json:"province,omitempty"`
	District        string  `json:"district,omitempty"`
	Subdistrict     string  `json:"subdistrict,omitempty"`
	AddressLine     string  `json:"addressLine,omitempty"`
	SubmittedAt     string  `json:"submittedAt,omitempty"`
}

// RescueRequestCurrentState คือ current state ของคำร้องจาก RescueRequest Service.
type RescueRequestCurrentState struct {
	Status        string `json:"status,omitempty"`
	AssignedUnitID string `json:"assignedUnitId,omitempty"`
	AssignedAt    string `json:"assignedAt,omitempty"`
	LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
}

// RescueRequestDetail คือ response body จาก GET /v1/rescue-requests/{requestId}.
type RescueRequestDetail struct {
	Master       RescueRequestMaster       `json:"master"`
	CurrentState RescueRequestCurrentState `json:"currentState"`
}

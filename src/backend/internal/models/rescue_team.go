package models

// RescueTeamLocation คือ sub-struct ของ location จาก RescueTeam Service.
type RescueTeamLocation struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	Source    string  `json:"source,omitempty"`
}

// RescueTeamDetail คือ response body จาก GET /v1/teams/{team_id}.
// อ้างอิง: Sync Contract #2
type RescueTeamDetail struct {
	TeamID       string             `json:"team_id"`
	TeamName     string             `json:"team_name"`
	TeamType     string             `json:"team_type"`
	Status       string             `json:"status"`
	Location     RescueTeamLocation `json:"location"`
	Capabilities []string           `json:"capabilities"`
	Equipment    []string           `json:"equipment,omitempty"`
	UpdatedAt    string             `json:"updated_at,omitempty"`
}

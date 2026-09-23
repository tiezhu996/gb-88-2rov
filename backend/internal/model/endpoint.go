package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ConditionRule is a conditional response rule evaluated against query/body fields.
type ConditionRule struct {
	Field        string `json:"field"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
	ResponseBody string `json:"responseBody"`
	StatusCode   int    `json:"statusCode"`
}

// MockAPI is a configurable mock endpoint under a project.
//
// The top-level path/method/... columns hold the published ("live")
// configuration that the public mock engine serves. When an endpoint is
// edited, the new values are instead stored as a draft snapshot in the
// Draft* columns and only take over after Publish copies them back.
type MockAPI struct {
	ID        uint `gorm:"primaryKey" json:"_id,string"`
	ProjectID uint `gorm:"index;not null" json:"projectId,string"`
	// Published configuration served to external requests.
	Path              string     `gorm:"size:255;not null" json:"path"`
	Method            string     `gorm:"size:16;not null" json:"method"`
	StatusCode        int        `gorm:"not null;default:200" json:"statusCode"`
	ResponseBody      string     `gorm:"type:text" json:"responseBody"`
	ResponseHeadersJS string     `gorm:"column:response_headers;type:text" json:"-"`
	Delay             int        `gorm:"not null;default:0" json:"delay"`
	ConditionsJS      string     `gorm:"column:conditions;type:text" json:"-"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`

	// Unpublished draft snapshot; ignored by the mock engine until published.
	HasDraft               bool       `gorm:"not null;default:false" json:"hasDraft"`
	DraftPath              string     `gorm:"size:255" json:"draftPath,omitempty"`
	DraftMethod            string     `gorm:"size:16" json:"draftMethod,omitempty"`
	DraftStatusCode        int        `json:"draftStatusCode,omitempty"`
	DraftResponseBody      string     `gorm:"type:text" json:"draftResponseBody,omitempty"`
	DraftResponseHeadersJS string     `gorm:"column:draft_response_headers;type:text" json:"-"`
	DraftDelay             int        `json:"draftDelay,omitempty"`
	DraftConditionsJS      string     `gorm:"column:draft_conditions;type:text" json:"-"`
	DraftSavedAt           *time.Time `json:"draftSavedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"-"`

	// Computed fields (not persisted).
	ResponseHeaders      map[string]string `gorm:"-" json:"responseHeaders"`
	Conditions           []ConditionRule   `gorm:"-" json:"conditions"`
	DraftResponseHeaders map[string]string `gorm:"-" json:"draftResponseHeaders,omitempty"`
	DraftConditions      []ConditionRule   `gorm:"-" json:"draftConditions,omitempty"`
}

// ClearDraft resets every draft column so the endpoint is "clean" again.
func (a *MockAPI) ClearDraft() {
	a.HasDraft = false
	a.DraftPath = ""
	a.DraftMethod = ""
	a.DraftStatusCode = 0
	a.DraftResponseBody = ""
	a.DraftResponseHeaders = nil
	a.DraftResponseHeadersJS = ""
	a.DraftDelay = 0
	a.DraftConditions = nil
	a.DraftConditionsJS = ""
	a.DraftSavedAt = nil
}

// BeforeSave serializes computed header/condition fields into JSON columns.
func (a *MockAPI) BeforeSave(_ *gorm.DB) error {
	if a.ResponseHeaders != nil {
		b, err := json.Marshal(a.ResponseHeaders)
		if err != nil {
			return err
		}
		a.ResponseHeadersJS = string(b)
	}
	if a.Conditions != nil {
		b, err := json.Marshal(a.Conditions)
		if err != nil {
			return err
		}
		a.ConditionsJS = string(b)
	}
	if a.DraftResponseHeaders != nil {
		b, err := json.Marshal(a.DraftResponseHeaders)
		if err != nil {
			return err
		}
		a.DraftResponseHeadersJS = string(b)
	}
	if a.DraftConditions != nil {
		b, err := json.Marshal(a.DraftConditions)
		if err != nil {
			return err
		}
		a.DraftConditionsJS = string(b)
	}
	return nil
}

// AfterFind restores computed fields from JSON columns.
func (a *MockAPI) AfterFind(_ *gorm.DB) error {
	a.ResponseHeaders = map[string]string{}
	if a.ResponseHeadersJS != "" {
		_ = json.Unmarshal([]byte(a.ResponseHeadersJS), &a.ResponseHeaders)
	}
	a.Conditions = []ConditionRule{}
	if a.ConditionsJS != "" {
		_ = json.Unmarshal([]byte(a.ConditionsJS), &a.Conditions)
	}
	if a.HasDraft {
		a.DraftResponseHeaders = map[string]string{}
		if a.DraftResponseHeadersJS != "" {
			_ = json.Unmarshal([]byte(a.DraftResponseHeadersJS), &a.DraftResponseHeaders)
		}
		a.DraftConditions = []ConditionRule{}
		if a.DraftConditionsJS != "" {
			_ = json.Unmarshal([]byte(a.DraftConditionsJS), &a.DraftConditions)
		}
	}
	return nil
}

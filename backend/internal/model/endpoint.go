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

// EndpointDraft is an unpublished snapshot of an endpoint's editable config.
// The live columns of MockAPI keep serving traffic until the draft is published.
type EndpointDraft struct {
	Path            string            `json:"path"`
	Method          string            `json:"method"`
	StatusCode      int               `json:"statusCode"`
	ResponseBody    string            `json:"responseBody"`
	ResponseHeaders map[string]string `json:"responseHeaders"`
	Delay           int               `json:"delay"`
	Conditions      []ConditionRule   `json:"conditions"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// MockAPI is a configurable mock endpoint under a project.
type MockAPI struct {
	ID                uint              `gorm:"primaryKey" json:"_id,string"`
	ProjectID         uint              `gorm:"index;not null" json:"projectId,string"`
	Path              string            `gorm:"size:255;not null" json:"path"`
	Method            string            `gorm:"size:16;not null" json:"method"`
	StatusCode        int               `gorm:"not null;default:200" json:"statusCode"`
	ResponseBody      string            `gorm:"type:text" json:"responseBody"`
	ResponseHeadersJS string            `gorm:"column:response_headers;type:text" json:"-"`
	Delay             int               `gorm:"not null;default:0" json:"delay"`
	ConditionsJS      string            `gorm:"column:conditions;type:text" json:"-"`
	HasDraft          bool              `gorm:"column:has_draft;not null;default:0" json:"hasDraft"`
	DraftJS           string            `gorm:"column:draft;type:text" json:"-"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"-"`

	// Computed fields (not persisted).
	ResponseHeaders map[string]string `gorm:"-" json:"responseHeaders"`
	Conditions      []ConditionRule   `gorm:"-" json:"conditions"`
	Draft           *EndpointDraft    `gorm:"-" json:"draft,omitempty"`
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
	// Publishing clears the draft snapshot; saving with a populated Draft keeps it.
	if a.HasDraft && a.Draft != nil {
		b, err := json.Marshal(a.Draft)
		if err != nil {
			return err
		}
		a.DraftJS = string(b)
	} else if !a.HasDraft {
		a.DraftJS = ""
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
	if a.HasDraft && a.DraftJS != "" {
		var d EndpointDraft
		if err := json.Unmarshal([]byte(a.DraftJS), &d); err == nil {
			a.Draft = &d
		}
	}
	return nil
}

package repository

import (
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// EndpointRepository persists mock endpoints.
type EndpointRepository struct {
	db *gorm.DB
}

// NewEndpointRepository builds an EndpointRepository.
func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

// Create inserts a mock endpoint.
func (r *EndpointRepository) Create(e *model.MockAPI) error {
	if err := r.db.Create(e).Error; err != nil {
		return fmt.Errorf("create endpoint: %w", err)
	}
	return nil
}

// ListByProject returns all endpoints of a project.
func (r *EndpointRepository) ListByProject(projectID uint) ([]model.MockAPI, error) {
	var endpoints []model.MockAPI
	if err := r.db.Where("project_id = ?", projectID).Order("id ASC").Find(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	return endpoints, nil
}

// FindByID loads an endpoint by primary key.
func (r *EndpointRepository) FindByID(id uint) (*model.MockAPI, error) {
	var e model.MockAPI
	if err := r.db.First(&e, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find endpoint by id: %w", err)
	}
	return &e, nil
}

// SaveDraft stores an unpublished draft snapshot without touching the live config.
func (r *EndpointRepository) SaveDraft(id uint, draft *model.EndpointDraft) error {
	b, err := json.Marshal(draft)
	if err != nil {
		return fmt.Errorf("marshal endpoint draft: %w", err)
	}
	if err := r.db.Model(&model.MockAPI{}).Where("id = ?", id).
		Updates(map[string]any{"has_draft": true, "draft": string(b)}).Error; err != nil {
		return fmt.Errorf("save endpoint draft: %w", err)
	}
	return nil
}

// PublishDraft copies the draft snapshot into the live columns and clears it,
// atomically, so requests only ever see the old or the new version.
func (r *EndpointRepository) PublishDraft(id uint, draft *model.EndpointDraft) error {
	headersJS, err := json.Marshal(draft.ResponseHeaders)
	if err != nil {
		return fmt.Errorf("marshal draft headers: %w", err)
	}
	conditionsJS, err := json.Marshal(draft.Conditions)
	if err != nil {
		return fmt.Errorf("marshal draft conditions: %w", err)
	}
	err = r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.MockAPI{}).Where("id = ?", id).Updates(map[string]any{
			"path":             draft.Path,
			"method":           draft.Method,
			"status_code":      draft.StatusCode,
			"response_body":    draft.ResponseBody,
			"response_headers": string(headersJS),
			"delay":            draft.Delay,
			"conditions":       string(conditionsJS),
			"has_draft":        false,
			"draft":            nil,
		})
		if res.Error != nil {
			return fmt.Errorf("publish endpoint draft: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// DiscardDraft drops the unpublished draft snapshot and keeps the live config.
func (r *EndpointRepository) DiscardDraft(id uint) error {
	res := r.db.Model(&model.MockAPI{}).Where("id = ? AND has_draft = ?", id, true).
		Updates(map[string]any{"has_draft": false, "draft": nil})
	if res.Error != nil {
		return fmt.Errorf("discard endpoint draft: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an endpoint.
func (r *EndpointRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.MockAPI{}, id).Error; err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}

// Match finds the first endpoint matching method and path (with :param support).
func (r *EndpointRepository) Match(projectID uint, method, path string) (*model.MockAPI, map[string]string, error) {
	endpoints, err := r.ListByProject(projectID)
	if err != nil {
		return nil, nil, err
	}
	for i := range endpoints {
		params, ok := matchPath(endpoints[i].Path, path)
		if ok && equalFold(endpoints[i].Method, method) {
			return &endpoints[i], params, nil
		}
	}
	return nil, nil, ErrNotFound
}

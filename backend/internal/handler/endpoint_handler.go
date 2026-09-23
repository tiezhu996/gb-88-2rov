package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/middleware"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// EndpointHandler exposes mock endpoint management endpoints.
type EndpointHandler struct {
	svc    *service.EndpointService
	logger *slog.Logger
}

// NewEndpointHandler builds an EndpointHandler.
func NewEndpointHandler(svc *service.EndpointService, logger *slog.Logger) *EndpointHandler {
	return &EndpointHandler{svc: svc, logger: logger}
}

// List handles GET /projects/:projectId/apis.
func (h *EndpointHandler) List(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	apis, err := h.svc.List(projectID, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, apis)
}

// Get handles GET /projects/:projectId/apis/:id.
func (h *EndpointHandler) Get(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	api, err := h.svc.Get(projectID, id, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// Create handles POST /projects/:projectId/apis.
func (h *EndpointHandler) Create(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.EndpointRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	api, err := h.svc.Create(projectID, middleware.GetUserID(c), middleware.GetRole(c), req)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// SaveDraft handles PUT /projects/:projectId/apis/:id/draft.
func (h *EndpointHandler) SaveDraft(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.EndpointRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	api, err := h.svc.SaveDraft(projectID, id, middleware.GetUserID(c), middleware.GetRole(c), req)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// PublishDraft handles POST /projects/:projectId/apis/:id/publish.
// The request body is optional: when present, the payload is saved as the
// draft before publishing, supporting a one-click "save & publish".
func (h *EndpointHandler) PublishDraft(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req *dto.EndpointRequest
	if c.Request.ContentLength != 0 {
		req = &dto.EndpointRequest{}
		if !util.BindAndValidate(c, req) {
			return
		}
	}
	api, err := h.svc.PublishDraft(projectID, id, middleware.GetUserID(c), middleware.GetRole(c), req)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// DiscardDraft handles DELETE /projects/:projectId/apis/:id/draft.
func (h *EndpointHandler) DiscardDraft(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	api, err := h.svc.DiscardDraft(projectID, id, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// Delete handles DELETE /projects/:projectId/apis/:id. Set force=true to
// delete an endpoint that still has an unpublished draft.
func (h *EndpointHandler) Delete(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	force := c.Query("force") == "true"
	if err := h.svc.Delete(projectID, id, middleware.GetUserID(c), middleware.GetRole(c), force); err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"deleted": true})
}

// ImportSwagger handles POST /projects/:projectId/swagger/import.
func (h *EndpointHandler) ImportSwagger(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.SwaggerImportRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	created, err := h.svc.ImportOpenAPI(projectID, middleware.GetUserID(c), middleware.GetRole(c), req.Document)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"created": created})
}

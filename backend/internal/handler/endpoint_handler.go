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

// Update handles PUT /projects/:projectId/apis/:id. The payload is stored as
// an unpublished draft; the live mock response keeps serving the old version.
func (h *EndpointHandler) Update(c *gin.Context) {
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
	api, err := h.svc.Update(projectID, id, middleware.GetUserID(c), middleware.GetRole(c), req)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// Publish handles POST /projects/:projectId/apis/:id/publish. The draft takes
// over the live mock response immediately.
func (h *EndpointHandler) Publish(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	api, err := h.svc.Publish(projectID, id, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, api)
}

// DiscardDraft handles POST /projects/:projectId/apis/:id/discard-draft. The
// draft is dropped and the live configuration stays as-is.
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

// Delete handles DELETE /projects/:projectId/apis/:id. Deleting an endpoint
// with an unpublished draft requires ?force=true.
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

package service

import (
	"errors"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestEndpointServiceCRUDAndAccess(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	endpoints := repository.NewEndpointRepository(db)
	svc := NewEndpointService(projects, endpoints, discardLogger())

	dev := createUser(t, db, "dev", RoleDev)
	other := createUser(t, db, "other", RoleDev)
	lead := createUser(t, db, "lead", RoleLead)
	project, err := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	req := dto.EndpointRequest{
		Path:            "/api/users/:id",
		Method:          "GET",
		StatusCode:      200,
		ResponseBody:    `{"ok":true}`,
		ResponseHeaders: map[string]string{"X-Demo": "1"},
	}
	created, err := svc.Create(project.ID, dev.ID, RoleDev, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.ResponseHeaders["X-Demo"] != "1" {
		t.Fatalf("Create = %+v", created)
	}

	t.Run("owner list", func(t *testing.T) {
		apis, err := svc.List(project.ID, dev.ID, RoleDev)
		if err != nil || len(apis) != 1 {
			t.Fatalf("List = %d, %v; want 1 api", len(apis), err)
		}
	})

	t.Run("lead list", func(t *testing.T) {
		apis, err := svc.List(project.ID, lead.ID, RoleLead)
		if err != nil || len(apis) != 1 {
			t.Fatalf("List lead = %d, %v", len(apis), err)
		}
	})

	t.Run("other dev forbidden", func(t *testing.T) {
		_, err := svc.List(project.ID, other.ID, RoleDev)
		if !errors.Is(err, constants.ErrForbidden) {
			t.Fatalf("List err = %v, want ErrForbidden", err)
		}
	})

	t.Run("update stores draft without touching live config", func(t *testing.T) {
		updated, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/api/users/:id", Method: "PUT", StatusCode: 200, ResponseBody: `{"updated":true}`,
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if !updated.HasDraft || updated.DraftMethod != "PUT" {
			t.Fatalf("Update draft = %+v, want hasDraft with draft method PUT", updated)
		}
		if updated.Method != "GET" || updated.ResponseBody != `{"ok":true}` {
			t.Fatalf("live config changed before publish: %+v", updated)
		}
	})

	t.Run("delete blocked while draft exists", func(t *testing.T) {
		err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev, false)
		var appErr *constants.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
			t.Fatalf("Delete err = %v, want 40900 conflict", err)
		}
	})

	t.Run("force delete drops draft too", func(t *testing.T) {
		if err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev, true); err != nil {
			t.Fatalf("Delete force: %v", err)
		}
		if _, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
		}
	})
}

func TestEndpointServiceDraftLifecycle(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	endpoints := repository.NewEndpointRepository(db)
	svc := NewEndpointService(projects, endpoints, discardLogger())
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/items", Method: "GET", StatusCode: 200, ResponseBody: `{"v":"live"}`,
		ResponseHeaders: map[string]string{"X-Version": "live"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	logs := repository.NewRequestLogRepository(db)
	engine := NewMockEngine(endpoints, logs, discardLogger())
	serve := func() *MockResult {
		t.Helper()
		res, err := engine.Handle(project.ID, "GET", "/api/items", nil, nil, nil)
		if err != nil {
			t.Fatalf("engine.Handle: %v", err)
		}
		return res
	}
	if res := serve(); res.StatusCode != 200 || res.Body != `{"v":"live"}` || res.Headers["X-Version"] != "live" {
		t.Fatalf("live response = %+v", res)
	}

	draftReq := dto.EndpointRequest{
		Path: "/api/items", Method: "GET", StatusCode: 202,
		ResponseBody: `{"v":"draft"}`, ResponseHeaders: map[string]string{"X-Version": "draft"},
	}

	// Saving a draft must not affect what the mock engine serves.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, draftReq); err != nil {
		t.Fatalf("Update draft: %v", err)
	}
	if res := serve(); res.StatusCode != 200 || res.Body != `{"v":"live"}` || res.Headers["X-Version"] != "live" {
		t.Fatalf("external request changed while draft pending: %+v", res)
	}
	matched, _, err := endpoints.Match(project.ID, "POST", "/api/items")
	if err == nil && matched.ID == created.ID {
		t.Fatal("draft method matched before publish; live config leaked")
	}
	live, _, err := endpoints.Match(project.ID, "GET", "/api/items")
	if err != nil || live.ResponseBody != `{"v":"live"}` {
		t.Fatalf("live endpoint not served while draft pending: %+v, %v", live, err)
	}

	// Publish: new config takes over immediately.
	published, err := svc.Publish(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if published.HasDraft || published.Method != "GET" || published.StatusCode != 202 ||
		published.ResponseHeaders["X-Version"] != "draft" || published.PublishedAt == nil {
		t.Fatalf("Publish = %+v", published)
	}
	if res := serve(); res.StatusCode != 202 || res.Body != `{"v":"draft"}` || res.Headers["X-Version"] != "draft" {
		t.Fatalf("draft did not take over after publish: %+v", res)
	}
	if _, _, err := endpoints.Match(project.ID, "POST", "/api/items"); err == nil {
		t.Fatal("unexpected match after publish")
	}

	// Publish/discard without a draft is an error.
	if _, err := svc.Publish(project.ID, created.ID, dev.ID, RoleDev); err == nil {
		t.Fatal("Publish without draft should fail")
	}
	if _, err := svc.DiscardDraft(project.ID, created.ID, dev.ID, RoleDev); err == nil {
		t.Fatal("DiscardDraft without draft should fail")
	}

	// A new draft can be discarded, returning to the current live version.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/items-v2", Method: "DELETE", StatusCode: 500, ResponseBody: `bad`,
	}); err != nil {
		t.Fatalf("Update second draft: %v", err)
	}
	discarded, err := svc.DiscardDraft(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil {
		t.Fatalf("DiscardDraft: %v", err)
	}
	if discarded.HasDraft || discarded.Method != "GET" || discarded.Path != "/api/items" {
		t.Fatalf("DiscardDraft = %+v, want published GET /api/items", discarded)
	}
	if res := serve(); res.StatusCode != 202 || res.Body != `{"v":"draft"}` {
		t.Fatalf("after discard engine served unexpected response: %+v", res)
	}

	// Saving a draft identical to the live config clears the draft flag.
	if _, err := svc.Update(project.ID, created.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/items", Method: "GET", StatusCode: 202,
		ResponseBody: `{"v":"draft"}`, ResponseHeaders: map[string]string{"X-Version": "draft"},
	}); err != nil {
		t.Fatalf("Update identical draft: %v", err)
	}
	fresh, err := svc.Get(project.ID, created.ID, dev.ID, RoleDev)
	if err != nil || fresh.HasDraft {
		t.Fatalf("identical draft should be cleared: %+v, %v", fresh, err)
	}
}

func TestEndpointServiceImportOpenAPI(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	endpoints := repository.NewEndpointRepository(db)
	svc := NewEndpointService(projects, endpoints, discardLogger())
	dev := createUser(t, db, "dev", RoleDev)
	project, _ := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")

	paths := map[string]any{}
	paths["/api/orders"] = map[string]any{
		"get": map[string]any{
			"responses": map[string]any{
				"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"example": map[string]any{"code": 0}}}},
			},
		},
		"post": map[string]any{
			"responses": map[string]any{"201": map[string]any{"content": map[string]any{"application/json": map[string]any{"example": map[string]any{"id": 1}}}}},
		},
	}
	doc := map[string]any{"openapi": "3.0.0", "paths": paths}
	created, err := svc.ImportOpenAPI(project.ID, dev.ID, RoleDev, doc)
	if err != nil {
		t.Fatalf("ImportOpenAPI: %v", err)
	}
	if created != 2 {
		t.Fatalf("ImportOpenAPI created = %d, want 2", created)
	}
}

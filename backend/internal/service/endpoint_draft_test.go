package service

import (
	"errors"
	"testing"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/repository"
)

func TestEndpointDraftLifecycle(t *testing.T) {
	db := newTestDB(t)
	projects := repository.NewProjectRepository(db)
	endpoints := repository.NewEndpointRepository(db)
	svc := NewEndpointService(projects, endpoints, discardLogger())

	dev := createUser(t, db, "dev", RoleDev)
	project, err := NewProjectService(projects, discardLogger()).Create("p", "d", dev.ID, "http://localhost:3119")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	created, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
		Path: "/api/users", Method: "GET", StatusCode: 200,
		ResponseBody:    `{"v":1}`,
		ResponseHeaders: map[string]string{"X-Live": "1"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	draftReq := dto.EndpointRequest{
		Path: "/api/v2/users", Method: "POST", StatusCode: 201,
		ResponseBody:    `{"v":2}`,
		ResponseHeaders: map[string]string{"X-Draft": "2"},
		Delay:           10,
	}

	t.Run("saving draft keeps live config", func(t *testing.T) {
		saved, err := svc.SaveDraft(project.ID, created.ID, dev.ID, RoleDev, draftReq)
		if err != nil {
			t.Fatalf("SaveDraft: %v", err)
		}
		if !saved.HasDraft || saved.Draft == nil {
			t.Fatalf("expected draft marker, got hasDraft=%v draft=%v", saved.HasDraft, saved.Draft)
		}
		if saved.Draft.Path != "/api/v2/users" || saved.Draft.StatusCode != 201 || saved.Draft.ResponseHeaders["X-Draft"] != "2" {
			t.Fatalf("draft payload mismatch: %+v", saved.Draft)
		}
		// Live columns stay on the old version and still match the old route.
		live, params, err := endpoints.Match(project.ID, "GET", "/api/users")
		if err != nil || live.ResponseBody != `{"v":1}` {
			t.Fatalf("live match = %+v, %v; want v1 response", live, err)
		}
		if len(params) != 0 {
			t.Fatalf("unexpected path params: %v", params)
		}
		if _, _, err := endpoints.Match(project.ID, "POST", "/api/v2/users"); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("draft route leaked into live matching: %v", err)
		}
	})

	t.Run("delete blocked while draft unpublished", func(t *testing.T) {
		err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev, false)
		var appErr *constants.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeConflict {
			t.Fatalf("Delete err = %v, want conflict AppError", err)
		}
	})

	t.Run("discard returns to live version", func(t *testing.T) {
		reverted, err := svc.DiscardDraft(project.ID, created.ID, dev.ID, RoleDev)
		if err != nil {
			t.Fatalf("DiscardDraft: %v", err)
		}
		if reverted.HasDraft || reverted.Draft != nil {
			t.Fatalf("draft should be cleared: hasDraft=%v", reverted.HasDraft)
		}
	})

	t.Run("publish takes over live response", func(t *testing.T) {
		if _, err := svc.SaveDraft(project.ID, created.ID, dev.ID, RoleDev, draftReq); err != nil {
			t.Fatalf("SaveDraft: %v", err)
		}
		published, err := svc.PublishDraft(project.ID, created.ID, dev.ID, RoleDev, nil)
		if err != nil {
			t.Fatalf("PublishDraft: %v", err)
		}
		if published.HasDraft || published.Draft != nil {
			t.Fatalf("draft should be cleared after publish")
		}
		if published.Path != "/api/v2/users" || published.Method != "POST" || published.StatusCode != 201 {
			t.Fatalf("live config not updated: %+v", published)
		}
		if published.ResponseHeaders["X-Draft"] != "2" || published.Delay != 10 {
			t.Fatalf("published headers/delay mismatch: %+v", published)
		}
		if _, _, err := endpoints.Match(project.ID, "GET", "/api/users"); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("old route still served after publish")
		}
		live, _, err := endpoints.Match(project.ID, "POST", "/api/v2/users")
		if err != nil || live.ResponseBody != `{"v":2}` {
			t.Fatalf("new route not live: %+v, %v", live, err)
		}
	})

	t.Run("publish without draft errors", func(t *testing.T) {
		if _, err := svc.PublishDraft(project.ID, created.ID, dev.ID, RoleDev, nil); err == nil {
			t.Fatal("PublishDraft without draft should fail")
		}
	})

	t.Run("publish with payload saves and publishes atomically", func(t *testing.T) {
		req := dto.EndpointRequest{
			Path: "/api/v3/users", Method: "GET", StatusCode: 200, ResponseBody: `{"v":3}`,
		}
		published, err := svc.PublishDraft(project.ID, created.ID, dev.ID, RoleDev, &req)
		if err != nil {
			t.Fatalf("PublishDraft with payload: %v", err)
		}
		if published.Path != "/api/v3/users" || published.ResponseBody != `{"v":3}` || published.HasDraft {
			t.Fatalf("one-click publish mismatch: %+v", published)
		}
	})

	t.Run("delete succeeds after draft cleared", func(t *testing.T) {
		if err := svc.Delete(project.ID, created.ID, dev.ID, RoleDev, false); err != nil {
			t.Fatalf("Delete after publish: %v", err)
		}
	})

	t.Run("force delete removes endpoint with draft", func(t *testing.T) {
		other, err := svc.Create(project.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/api/other", Method: "GET", StatusCode: 200,
		})
		if err != nil {
			t.Fatalf("Create other: %v", err)
		}
		if _, err := svc.SaveDraft(project.ID, other.ID, dev.ID, RoleDev, dto.EndpointRequest{
			Path: "/api/other", Method: "GET", StatusCode: 500,
		}); err != nil {
			t.Fatalf("SaveDraft: %v", err)
		}
		if err := svc.Delete(project.ID, other.ID, dev.ID, RoleDev, true); err != nil {
			t.Fatalf("force Delete: %v", err)
		}
		if _, err := svc.Get(project.ID, other.ID, dev.ID, RoleDev); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("Get after force delete err = %v, want ErrNotFound", err)
		}
	})
}

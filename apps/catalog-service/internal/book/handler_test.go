package book

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fizzisme/catalog-service/internal/logger"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	gotList    ListInput
	gotSlug    string
	gotLimit   int
	listCalled bool
	gotSlugs   []string
	err        error
}

func (f *fakeService) List(_ context.Context, in ListInput) (Page, error) {
	f.listCalled = true
	f.gotList = in
	if f.err != nil {
		return Page{}, f.err
	}
	return Page{Items: []Book{{Slug: "a", Title: "A", PriceVND: 100}}, Total: 42, Page: 2, PageSize: PageSize}, nil
}

func (f *fakeService) Get(_ context.Context, slug string) (Book, error) {
	f.gotSlug = slug
	if f.err != nil {
		return Book{}, f.err
	}
	return Book{Slug: slug, Title: "A", PriceVND: 100}, nil
}

func (f *fakeService) Related(_ context.Context, slug string, limit int) ([]Book, error) {
	f.gotSlug, f.gotLimit = slug, limit
	if f.err != nil {
		return nil, f.err
	}
	return []Book{{Slug: "b"}}, nil
}

func (f *fakeService) BySlugs(_ context.Context, slugs []string) ([]Book, error) {
	f.gotSlugs = slugs
	if f.err != nil {
		return nil, f.err
	}
	return []Book{{Slug: "a"}, {Slug: "b"}}, nil
}

func serve(t *testing.T, svc service, target string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	NewHandler(svc).Register(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func TestListOK(t *testing.T) {
	svc := &fakeService{}

	w := serve(t, svc, "/api/v1/books?subject=history&rare=true&sort=price-asc&q=holmes&page=2")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}

	want := ListInput{Subject: "history", Rare: true, Sort: "price-asc", Q: "holmes", Page: 2}
	if svc.gotList != want {
		t.Errorf("input = %+v, want %+v", svc.gotList, want)
	}

	for _, field := range []string{`"items":[`, `"total":42`, `"page":2`, `"pageSize":8`, `"priceVnd":100`} {
		if !strings.Contains(w.Body.String(), field) {
			t.Errorf("body %s is missing %s", w.Body, field)
		}
	}
}

func TestListRejectsBadQuery(t *testing.T) {
	for _, target := range []string{
		"/api/v1/books?sort=abc",
		"/api/v1/books?rare=maybe",
		"/api/v1/books?page=-1",
		"/api/v1/books?page=x",
		"/api/v1/books?q=" + strings.Repeat("a", 101),
	} {
		svc := &fakeService{}

		w := serve(t, svc, target)

		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", target, w.Code)
		}
		if svc.listCalled {
			t.Errorf("%s: service was called", target)
		}
	}
}

func TestGet(t *testing.T) {
	svc := &fakeService{}

	w := serve(t, svc, "/api/v1/books/some-slug")

	if w.Code != http.StatusOK || svc.gotSlug != "some-slug" {
		t.Errorf("status = %d, slug = %q", w.Code, svc.gotSlug)
	}
}

func TestNotFound(t *testing.T) {
	for _, target := range []string{"/api/v1/books/nope", "/api/v1/books/nope/related"} {
		w := serve(t, &fakeService{err: ErrNotFound}, target)

		if w.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", target, w.Code)
		}
	}
}

func TestInternalErrorDoesNotLeak(t *testing.T) {
	err := errors.New("mssql: login failed for user 'sa' on 10.0.0.5")

	for _, target := range []string{"/api/v1/books", "/api/v1/books/x", "/api/v1/books/x/related"} {
		w := serve(t, &fakeService{err: err}, target)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want 500", target, w.Code)
		}
		if strings.Contains(w.Body.String(), "mssql") || strings.Contains(w.Body.String(), "10.0.0.5") {
			t.Errorf("%s: body leaks the error: %s", target, w.Body)
		}
	}
}

func TestHandlerRelatedLimit(t *testing.T) {
	svc := &fakeService{}

	w := serve(t, svc, "/api/v1/books/x/related?limit=6")

	if w.Code != http.StatusOK || svc.gotLimit != 6 {
		t.Errorf("status = %d, limit = %d", w.Code, svc.gotLimit)
	}

	if w := serve(t, &fakeService{}, "/api/v1/books/x/related?limit=0"); w.Code != http.StatusOK {
		t.Errorf("limit=0 should fall back to the default, got %d", w.Code)
	}

	if w := serve(t, &fakeService{}, "/api/v1/books/x/related?limit=-2"); w.Code != http.StatusBadRequest {
		t.Errorf("limit=-2: status = %d, want 400", w.Code)
	}
}

func TestListBySlugs(t *testing.T) {
	svc := &fakeService{}

	w := serve(t, svc, "/api/v1/books?slugs=a,%20b,c&subject=history&page=3")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	if !reflect.DeepEqual(svc.gotSlugs, []string{"a", "b", "c"}) {
		t.Errorf("slugs = %q, want [a b c]", svc.gotSlugs)
	}
	if svc.listCalled {
		t.Error("the other filters must be ignored when slugs is given")
	}
	if !strings.Contains(w.Body.String(), `"total":2`) || !strings.Contains(w.Body.String(), `"items":[`) {
		t.Errorf("body = %s", w.Body)
	}
}

func TestListBySlugsTooMany(t *testing.T) {
	slugs := strings.TrimSuffix(strings.Repeat("a,", MaxBatch+1), ",")
	svc := &fakeService{}

	w := serve(t, svc, "/api/v1/books?slugs="+slugs)

	if w.Code != http.StatusBadRequest || svc.gotSlugs != nil {
		t.Errorf("status = %d, service called = %v", w.Code, svc.gotSlugs != nil)
	}
}

func TestListBySlugsError(t *testing.T) {
	w := serve(t, &fakeService{err: errors.New("mssql: boom")}, "/api/v1/books?slugs=a")

	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "mssql") {
		t.Errorf("status = %d, body = %s", w.Code, w.Body)
	}
}

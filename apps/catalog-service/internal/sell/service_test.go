package sell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

type fakeStore struct {
	created []Request
	deleted []int64
}

func (f *fakeStore) Create(_ context.Context, in CreateInput) (Request, error) {
	r := Request{ID: int64(len(f.created) + 41), Title: in.Title, Status: StatusPendingReview}
	f.created = append(f.created, r)
	return r, nil
}

func (f *fakeStore) Delete(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeStore) List(context.Context, int) ([]Request, error) { return nil, nil }

func (f *fakeStore) Photo(context.Context, int64, int) (Photo, error) {
	return Photo{}, ErrPhotoNotFound
}

type fakePhotos struct {
	put     []string
	deleted []string
	failAt  int
}

func (f *fakePhotos) Put(_ context.Context, name, _ string, body io.ReadSeekCloser, _ map[string]string) error {
	if f.failAt > 0 && len(f.put)+1 == f.failAt {
		return errors.New("storage down")
	}
	f.put = append(f.put, name)
	return nil
}

func (f *fakePhotos) Delete(_ context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func (f *fakePhotos) Open(context.Context, string) (io.ReadCloser, string, int64, error) {
	return nil, "", 0, ErrPhotoNotFound
}

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func fileHeaders(t *testing.T, files ...[]byte) []*multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, content := range files {
		part, err := w.CreateFormFile("photos", "photo")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(content)
	}
	w.Close()

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["photos"]
}

func validInput() CreateInput {
	return CreateInput{
		Title: "Rebecca", Author: "Daphne du Maurier", Binding: "Hardcover", Grade: "Good",
		Faults: "Jacket torn", Name: "Reader", Email: "reader@example.com",
	}
}

func TestUploadsRejectsNonImages(t *testing.T) {
	_, err := Uploads(fileHeaders(t, []byte("plain text, not a picture")))
	if !errors.Is(err, ErrNotAnImage) {
		t.Fatalf("err = %v, want ErrNotAnImage", err)
	}
}

func TestUploadsRejectsTooMany(t *testing.T) {
	files := make([][]byte, MaxPhotos+1)
	for i := range files {
		files[i] = pngHeader
	}
	_, err := Uploads(fileHeaders(t, files...))
	if !errors.Is(err, ErrTooManyPhotos) {
		t.Fatalf("err = %v, want ErrTooManyPhotos", err)
	}
}

func TestCreateStoresPhotosUnderTheRequestID(t *testing.T) {
	uploads, err := Uploads(fileHeaders(t, pngHeader, pngHeader))
	if err != nil {
		t.Fatal(err)
	}
	store, photos := &fakeStore{}, &fakePhotos{}

	req, err := NewService(store, photos).Create(context.Background(), validInput(), uploads)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"41/1.png", "41/2.png"}
	if req.ID != 41 || len(photos.put) != 2 || photos.put[0] != want[0] || photos.put[1] != want[1] {
		t.Fatalf("id %d, put %v, want %v", req.ID, photos.put, want)
	}
}

func TestCreateUndoesEverythingWhenAPhotoFails(t *testing.T) {
	uploads, err := Uploads(fileHeaders(t, pngHeader, pngHeader))
	if err != nil {
		t.Fatal(err)
	}
	store, photos := &fakeStore{}, &fakePhotos{failAt: 2}

	_, err = NewService(store, photos).Create(context.Background(), validInput(), uploads)
	if err == nil {
		t.Fatal("want an error")
	}
	if len(store.deleted) != 1 || store.deleted[0] != 41 {
		t.Fatalf("deleted requests %v, want [41]", store.deleted)
	}
	if len(photos.deleted) != 1 || photos.deleted[0] != "41/1.png" {
		t.Fatalf("deleted photos %v, want [41/1.png]", photos.deleted)
	}
}

func TestCreateRefusesPhotosWithoutStorage(t *testing.T) {
	uploads, err := Uploads(fileHeaders(t, pngHeader))
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewService(&fakeStore{}, nil).Create(context.Background(), validInput(), uploads)
	if !errors.Is(err, ErrPhotosDisabled) {
		t.Fatalf("err = %v, want ErrPhotosDisabled", err)
	}
}

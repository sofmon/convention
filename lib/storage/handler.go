package storage

import (
	"io"
	"net/http"
	"strings"

	convAPI "github.com/sofmon/convention/lib/api"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

// NewHandler creates a convAPI.Raw handler for storage operations.
// The handler dispatches to Save/Load/Delete/Exists based on HTTP method:
//   - PUT: Save file
//   - GET: Load file
//   - DELETE: Delete file
//   - HEAD: Check if file exists
//
// Usage in service API:
//
//	type API struct {
//	    Storage convAPI.Raw `api:"* /asset/v1/storage/{any...}"`
//	}
//	api := &API{Storage: storage.NewHandler(s,"/asset/v1/storage")}
func NewHandler(s *Storage, prefix string) convAPI.Raw {
	if prefix != "" {
		prefix = "/" + strings.Trim(prefix, "/")
	}

	return convAPI.NewRaw(func(ctx convCtx.Context, w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if prefix != "" && strings.HasPrefix(path, prefix) {
			path = strings.Trim(path[len(prefix):], "/")
		}

		switch r.Method {
		case http.MethodPut:
			handleSave(ctx, s, path, w, r)
		case http.MethodGet:
			handleLoad(ctx, s, path, w, r)
		case http.MethodDelete:
			handleDelete(ctx, s, path, w, r)
		case http.MethodHead:
			handleExists(ctx, s, path, w, r)
		default:
			convAPI.ServeError(ctx, w, http.StatusMethodNotAllowed,
				convAPI.ErrorCodeBadRequest, "method not allowed", nil)
		}
	})
}

func handleSave(ctx convCtx.Context, s *Storage, path string, w http.ResponseWriter, r *http.Request) {
	ctx = ctx.WithScope("storage.handleSave", "path", path)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		convAPI.ServeError(ctx, w, http.StatusBadRequest, convAPI.ErrorCodeBadRequest, "failed to read body", err)
		return
	}

	if err := s.Save(ctx, path, data); err != nil {
		convAPI.ServeError(ctx, w, http.StatusInternalServerError, convAPI.ErrorCodeInternalError, "failed to save", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleLoad(ctx convCtx.Context, s *Storage, path string, w http.ResponseWriter, r *http.Request) {
	ctx = ctx.WithScope("storage.handleLoad", "path", path)

	data, err := s.Load(ctx, path)
	if err != nil {
		convAPI.ServeError(ctx, w, http.StatusNotFound, convAPI.ErrorCodeNotFound, "file not found", err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleDelete(ctx convCtx.Context, s *Storage, path string, w http.ResponseWriter, r *http.Request) {
	ctx = ctx.WithScope("storage.handleDelete", "path", path)

	if err := s.Delete(ctx, path); err != nil {
		convAPI.ServeError(ctx, w, http.StatusInternalServerError, convAPI.ErrorCodeInternalError, "failed to delete", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleExists(ctx convCtx.Context, s *Storage, path string, w http.ResponseWriter, r *http.Request) {
	ctx = ctx.WithScope("storage.handleExists", "path", path)

	exists, err := s.Exists(ctx, path)
	if err != nil {
		convAPI.ServeError(ctx, w, http.StatusInternalServerError, convAPI.ErrorCodeInternalError, "failed to check", err)
		return
	}

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

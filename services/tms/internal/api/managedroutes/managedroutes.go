package managedroutes

import (
	"io"
	"net/http"
	"strconv"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

const ProblemJSON = "application/problem+json"

type Contributor interface {
	RouteManaged(mux *http.ServeMux)
}

type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func WriteProblem(w http.ResponseWriter, l *zap.Logger, status int, detail string) {
	w.Header().Set("Content-Type", ProblemJSON)
	w.WriteHeader(status)
	body, err := sonic.Marshal(Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
	if err == nil {
		_, err = w.Write(body)
	}
	if err != nil {
		l.Warn("cannot write a problem to the managed listener's caller", zap.Error(err))
	}
}

func WriteJSON(w http.ResponseWriter, l *zap.Logger, status int, v any) {
	body, err := sonic.Marshal(v)
	if err != nil {
		l.Error("cannot encode an answer to the managed listener's caller", zap.Error(err))
		WriteProblem(w, l, http.StatusInternalServerError, "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		l.Warn("cannot write an answer to the managed listener's caller", zap.Error(err))
	}
}

func ReadBody(w http.ResponseWriter, r *http.Request, l *zap.Logger, limit int64) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		WriteProblem(
			w,
			l,
			http.StatusRequestEntityTooLarge,
			"the body is larger than "+strconv.FormatInt(limit, 10)+" bytes",
		)
		return nil, false
	}
	return body, true
}

func Refuse(l *zap.Logger, detail string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteProblem(w, l, http.StatusConflict, detail)
	}
}

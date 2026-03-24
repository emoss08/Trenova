package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

var (
	TestUserID = pulid.MustNew("usr")
	TestOrgID  = pulid.MustNew("org")
	TestBuID   = pulid.MustNew("bu")
)

func init() {
	gin.SetMode(gin.TestMode)
}

type GinTestContext struct {
	Context  *gin.Context
	Recorder *httptest.ResponseRecorder
	Engine   *gin.Engine
}

func NewGinTestContext() *GinTestContext {
	recorder := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	return &GinTestContext{
		Context:  c,
		Recorder: recorder,
		Engine:   engine,
	}
}

func (g *GinTestContext) WithMethod(method string) *GinTestContext {
	g.Context.Request.Method = method
	return g
}

func (g *GinTestContext) WithPath(path string) *GinTestContext {
	g.Context.Request.URL.Path = path
	return g
}

func (g *GinTestContext) WithQuery(params map[string]string) *GinTestContext {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	g.Context.Request.URL.RawQuery = q.Encode()
	return g
}

func (g *GinTestContext) WithQueryValues(params map[string][]string) *GinTestContext {
	q := url.Values{}
	for k, values := range params {
		for _, v := range values {
			q.Add(k, v)
		}
	}
	g.Context.Request.URL.RawQuery = q.Encode()
	return g
}

func (g *GinTestContext) WithHeader(key, value string) *GinTestContext {
	g.Context.Request.Header.Set(key, value)
	return g
}

func (g *GinTestContext) WithHeaders(headers map[string]string) *GinTestContext {
	for k, v := range headers {
		g.Context.Request.Header.Set(k, v)
	}
	return g
}

func (g *GinTestContext) WithJSONBody(body any) *GinTestContext {
	data, _ := json.Marshal(body)
	g.Context.Request.Body = io.NopCloser(bytes.NewBuffer(data))
	g.Context.Request.Header.Set("Content-Type", "application/json")
	return g
}

func (g *GinTestContext) WithBody(body string) *GinTestContext {
	g.Context.Request.Body = io.NopCloser(strings.NewReader(body))
	return g
}

func (g *GinTestContext) WithContextValue(key string, value any) *GinTestContext {
	g.Context.Set(key, value)
	return g
}

func (g *GinTestContext) WithAuthContext(userID, orgID, buID pulid.ID) *GinTestContext {
	authctx.SetAuthContext(g.Context, userID, buID, orgID)
	g.Engine.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, userID, buID, orgID)
		c.Next()
	})
	return g
}

func (g *GinTestContext) WithDefaultAuthContext() *GinTestContext {
	return g.WithAuthContext(TestUserID, TestOrgID, TestBuID)
}

func (g *GinTestContext) WithParam(key, value string) *GinTestContext {
	g.Context.Params = append(g.Context.Params, gin.Param{Key: key, Value: value})
	return g
}

func (g *GinTestContext) ResponseBody() []byte {
	return g.Recorder.Body.Bytes()
}

func (g *GinTestContext) ResponseJSON(v any) error {
	return json.Unmarshal(g.Recorder.Body.Bytes(), v)
}

func (g *GinTestContext) ResponseCode() int {
	return g.Recorder.Code
}

func (g *GinTestContext) ResponseHeader(key string) string {
	return g.Recorder.Header().Get(key)
}

func (g *GinTestContext) Reset() *GinTestContext {
	g.Recorder = httptest.NewRecorder()
	g.Context, g.Engine = gin.CreateTestContext(g.Recorder)
	g.Context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return g
}

type MultipartFile struct {
	FieldName   string
	Filename    string
	ContentType string
	Data        []byte
}

func (g *GinTestContext) WithMultipartForm(
	fields map[string]string,
	files ...MultipartFile,
) *GinTestContext {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}

	for _, file := range files {
		part, err := writer.CreateFormFile(file.FieldName, file.Filename)
		if err != nil {
			continue
		}
		_, _ = part.Write(file.Data)
	}

	_ = writer.Close()

	g.Context.Request.Body = io.NopCloser(body)
	g.Context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	g.Context.Request.ContentLength = int64(body.Len())
	return g
}

func (g *GinTestContext) WithMultipartFormFiles(
	fields map[string]string,
	fieldName string,
	files []MultipartFile,
) *GinTestContext {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}

	for _, file := range files {
		part, err := writer.CreateFormFile(fieldName, file.Filename)
		if err != nil {
			continue
		}
		_, _ = part.Write(file.Data)
	}

	_ = writer.Close()

	g.Context.Request.Body = io.NopCloser(body)
	g.Context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	g.Context.Request.ContentLength = int64(body.Len())
	return g
}

package middleware

import (
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/request_detail_setting"

	"github.com/gin-gonic/gin"
)

type requestDetailResponseWriter struct {
	gin.ResponseWriter
	body           []byte
	bodySize       int64
	maxBodyBytes   int
	captureAll     bool
	contentChecked bool
	truncated      bool
	omittedReason  string
}

var _ gin.ResponseWriter = (*requestDetailResponseWriter)(nil)

func (w *requestDetailResponseWriter) Write(data []byte) (int, error) {
	written, err := w.ResponseWriter.Write(data)
	w.capture(data[:written])
	return written, err
}

func (w *requestDetailResponseWriter) WriteString(data string) (int, error) {
	written, err := w.ResponseWriter.WriteString(data)
	w.capture([]byte(data[:written]))
	return written, err
}

func (w *requestDetailResponseWriter) capture(data []byte) {
	if len(data) == 0 {
		return
	}
	w.bodySize += int64(len(data))
	if !w.captureAll && w.Status() < 400 {
		return
	}

	if !w.contentChecked {
		w.contentChecked = true
		contentEncoding := strings.ToLower(strings.TrimSpace(w.Header().Get("Content-Encoding")))
		if contentEncoding != "" && contentEncoding != "identity" {
			w.omittedReason = "content_encoded"
		}
		if !service.IsRequestDetailResponseBodyTypeSupported(w.Header().Get("Content-Type")) {
			w.omittedReason = "unsupported_content_type"
		}
	}
	if w.omittedReason != "" || w.truncated {
		return
	}

	remaining := w.maxBodyBytes - len(w.body)
	if remaining <= 0 {
		w.truncated = true
		return
	}
	if len(data) > remaining {
		w.body = append(w.body, data[:remaining]...)
		w.truncated = true
		return
	}
	w.body = append(w.body, data...)
}

// RequestDetailCapture must run after authentication and before routing. This
// keeps unauthenticated payloads out while still capturing distributor errors.
func RequestDetailCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := request_detail_setting.GetSetting()
		if settings.Mode == request_detail_setting.ModeNone {
			c.Next()
			return
		}

		responseWriter := &requestDetailResponseWriter{
			ResponseWriter: c.Writer,
			maxBodyBytes:   request_detail_setting.MaxResponseBodyBytes,
			captureAll:     settings.Mode == request_detail_setting.ModeAll,
		}
		c.Writer = responseWriter
		c.Next()

		bodySize := responseWriter.bodySize
		if size := responseWriter.Size(); size > 0 {
			bodySize = int64(size)
		}
		service.SetRequestDetailResponse(c, &service.RequestDetailResponseSnapshot{
			StatusCode:    responseWriter.Status(),
			ContentType:   responseWriter.Header().Get("Content-Type"),
			BodySize:      bodySize,
			Body:          responseWriter.body,
			Truncated:     responseWriter.truncated,
			OmittedReason: responseWriter.omittedReason,
		})
		service.CaptureRequestDetailFromContext(c)
	}
}

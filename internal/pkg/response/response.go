package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
)

type Body struct {
	RequestID string `json:"request_id"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Meta      *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func OK(c *gin.Context, data any) {
	OKMeta(c, data, nil)
}

func OKMeta(c *gin.Context, data any, meta *Meta) {
	c.JSON(http.StatusOK, Body{
		RequestID: requestID(c),
		Code:      0,
		Message:   "success",
		Data:      data,
		Meta:      meta,
	})
}

func Fail(c *gin.Context, err *apperr.AppError) {
	status := err.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	c.JSON(status, Body{
		RequestID: requestID(c),
		Code:      err.Code,
		Message:   err.Message,
		Data:      err.Fields,
	})
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return uuid.NewString()
}

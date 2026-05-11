package app

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func uid(c *gin.Context) uuid.UUID {
	return c.MustGet("user_id").(uuid.UUID)
}

func role(c *gin.Context) string {
	v, _ := c.Get("user_role")
	s, _ := v.(string)
	return s
}

func parseUUID(p string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(p))
}

func jbytes(v any) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	if string(b) == "null" {
		return []byte("{}")
	}
	return b
}

func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

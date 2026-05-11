package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
	"gorm.io/gorm"
)

type Params struct {
	Page     int
	PageSize int
	Offset   int
	Keyword  string
}

func FromQuery(c *gin.Context) Params {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if ps < 1 {
		ps = 20
	}
	if ps > 200 {
		ps = 200
	}
	return Params{
		Page:     p,
		PageSize: ps,
		Offset:   (p - 1) * ps,
		Keyword:  c.Query("keyword"),
	}
}

func Meta(p Params, total int64) *response.Meta {
	return &response.Meta{Page: p.Page, PageSize: p.PageSize, Total: total}
}

func ScopeKeyword(column string, kw string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if kw == "" {
			return db
		}
		like := "%" + kw + "%"
		return db.Where(column+" ILIKE ?", like)
	}
}

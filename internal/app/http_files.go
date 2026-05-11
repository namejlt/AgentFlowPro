package app

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/namejlt/AgentFlowPro/internal/model"
	"github.com/namejlt/AgentFlowPro/internal/pkg/apperr"
	"github.com/namejlt/AgentFlowPro/internal/pkg/response"
)

func (a *App) UploadFile(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, apperr.ErrBadRequest)
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	defer src.Close()

	baseDir := filepath.Join("data", "uploads")
	_ = os.MkdirAll(baseDir, 0o755)
	mt := fh.Header.Get("Content-Type")
	rec := model.UploadedFile{OwnerID: uid(c), StorageKey: "pending", OriginalName: fh.Filename, MimeType: &mt, SizeBytes: 0}
	if err := a.DB.Create(&rec).Error; err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	dstPath := filepath.Join(baseDir, rec.ID.String()+"_"+filepath.Base(fh.Filename))
	out, err := os.Create(dstPath)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	defer out.Close()
	n, err := io.Copy(out, src)
	if err != nil {
		response.Fail(c, apperr.ErrInternal)
		return
	}
	_ = a.DB.Model(&rec).Updates(map[string]any{"storage_key": dstPath, "size_bytes": n}).Error
	response.OK(c, gin.H{"id": rec.ID.String(), "path": dstPath, "size": n})
}

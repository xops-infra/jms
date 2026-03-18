package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"gorm.io/gorm"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/model"
)

const uploadSessionTTL = 24 * time.Hour

func uploadInit(c *gin.Context) {
	var req model.UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if req.Size <= 0 {
		c.String(http.StatusBadRequest, "size must be positive")
		return
	}
	if req.Host == "" || req.Path == "" {
		c.String(http.StatusBadRequest, "host and path required")
		return
	}
	if err := validateRemotePath(req.Path); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if req.Size > app.App.Config.Upload.MaxSize {
		c.String(http.StatusBadRequest, "size exceeds limit")
		return
	}
	if app.App.Config.Upload.Store.Type != "fs" {
		c.String(http.StatusNotImplemented, "store type not supported")
		return
	}
	chunkSize := req.ChunkSize
	if chunkSize <= 0 || chunkSize > app.App.Config.Upload.ChunkSize {
		chunkSize = app.App.Config.Upload.ChunkSize
	}
	chunkCount := int(math.Ceil(float64(req.Size) / float64(chunkSize)))
	if chunkCount <= 0 {
		chunkCount = 1
	}

	uploadID := uuid.NewString()
	sess := &model.UploadSession{
		ID:              uploadID,
		Host:            req.Host,
		SSHUser:         req.User,
		Path:            req.Path,
		Size:            req.Size,
		ChunkSize:       chunkSize,
		ChunkCount:      chunkCount,
		CompletedChunks: 0,
		SHA256:          req.SHA256,
		Status:          "init",
		ExpiresAt:       time.Now().Add(uploadSessionTTL),
	}
	if err := app.App.DBIo.CreateUploadSession(sess); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if err := os.MkdirAll(uploadDir(uploadID), 0755); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, model.UploadInitResponse{
		UploadID:  uploadID,
		ChunkSize: chunkSize,
		ExpiresAt: sess.ExpiresAt.Unix(),
	})
}

func uploadChunk(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		c.String(http.StatusBadRequest, "upload_id required")
		return
	}
	idxRaw := c.Query("index")
	idx, err := strconv.Atoi(idxRaw)
	if err != nil || idx < 0 {
		c.String(http.StatusBadRequest, "invalid index")
		return
	}
	if app.App.Config.Upload.Store.Type != "fs" {
		c.String(http.StatusNotImplemented, "store type not supported")
		return
	}

	sess, err := app.App.DBIo.GetUploadSession(uploadID)
	if err != nil {
		c.String(http.StatusNotFound, "upload session not found")
		return
	}
	if sess.Status == "aborted" || sess.Status == "completed" {
		c.String(http.StatusBadRequest, "invalid upload status")
		return
	}
	if time.Now().After(sess.ExpiresAt) {
		c.String(http.StatusBadRequest, "upload session expired")
		return
	}
	if idx >= sess.ChunkCount {
		c.String(http.StatusBadRequest, "index out of range")
		return
	}

	partPath := chunkPath(uploadID, idx)
	if _, err := os.Stat(partPath); err == nil {
		c.String(http.StatusOK, "ok")
		return
	}

	if err := os.MkdirAll(uploadDir(uploadID), 0755); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	limit := sess.ChunkSize
	if idx == sess.ChunkCount-1 {
		expected := sess.Size - int64(idx)*sess.ChunkSize
		if expected > 0 {
			limit = expected
		}
	}
	if limit <= 0 {
		limit = sess.ChunkSize
	}

	tmpPath := partPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()

	lr := io.LimitReader(c.Request.Body, limit+1)
	written, err := io.Copy(f, lr)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if written > limit {
		_ = os.Remove(tmpPath)
		c.String(http.StatusBadRequest, "chunk too large")
		return
	}
	if err := os.Rename(tmpPath, partPath); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	_ = app.App.DBIo.UpdateUploadSession(sess.ID, map[string]any{
		"completed_chunks": gorm.Expr("completed_chunks + ?", 1),
		"status":           "in_progress",
	})

	c.String(http.StatusOK, "ok")
}

func uploadComplete(c *gin.Context) {
	var req model.UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if app.App.Config.Upload.Store.Type != "fs" {
		c.String(http.StatusNotImplemented, "store type not supported")
		return
	}
	sess, err := app.App.DBIo.GetUploadSession(req.UploadID)
	if err != nil {
		c.String(http.StatusNotFound, "upload session not found")
		return
	}
	if sess.Status == "aborted" || sess.Status == "completed" {
		c.String(http.StatusBadRequest, "invalid upload status")
		return
	}
	if time.Now().After(sess.ExpiresAt) {
		c.String(http.StatusBadRequest, "upload session expired")
		return
	}
	if req.TotalChunks != sess.ChunkCount {
		c.String(http.StatusBadRequest, "chunk count mismatch")
		return
	}

	missing := missingChunks(sess.ID, sess.ChunkCount)
	if len(missing) > 0 {
		c.String(http.StatusBadRequest, fmt.Sprintf("missing chunks: %v", missing))
		return
	}

	v, ok := c.Get("auth_user")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	authUser := v.(model.User)
	if authUser.Username == nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}

	server, sshUsers, err := app.App.Sshd.SshdIO.GetSSHUsersByHostLive(sess.Host)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	sshUser, err := selectSSHUser(sshUsers, valueOrEmpty(sess.SSHUser), "")
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if err := app.App.Sshd.SshdIO.CheckPermission(fmt.Sprintf("%s@%s:%s", sshUser.UserName, sess.Host, sess.Path), authUser, model.Upload); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	_, upstream, err := sshd.NewSSHClient(*authUser.Username, *server, sshUser)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	defer upstream.Close()

	sftpClient, err := sftp.NewClient(upstream)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Create(sess.Path)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer remoteFile.Close()

	h := sha256.New()
	var writer io.Writer = remoteFile
	if req.SHA256 != nil && *req.SHA256 != "" {
		writer = io.MultiWriter(remoteFile, h)
	}

	buf := make([]byte, 256*1024)
	for i := 0; i < sess.ChunkCount; i++ {
		partPath := chunkPath(sess.ID, i)
		f, err := os.Open(partPath)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		_, err = io.CopyBuffer(writer, f, buf)
		f.Close()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.SHA256 != nil && *req.SHA256 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, *req.SHA256) {
			_ = sftpClient.Remove(sess.Path)
			c.String(http.StatusBadRequest, "sha256 mismatch")
			return
		}
	}

	_ = app.App.DBIo.UpdateUploadSession(sess.ID, map[string]any{
		"status": "completed",
	})
	_ = os.RemoveAll(uploadDir(sess.ID))

	if app.App.DBIo != nil {
		action := "upload"
		from := sess.ID
		to := fmt.Sprintf("%s:%s", sess.Host, sess.Path)
		client := c.ClientIP()
		_ = app.App.DBIo.AddScpRecord(&model.AddScpRecordRequest{
			Action: &action,
			From:   &from,
			To:     &to,
			User:   authUser.Username,
			Client: &client,
		})
	}

	c.String(http.StatusOK, "ok")
}

func uploadAbort(c *gin.Context) {
	var req model.UploadAbortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if app.App.Config.Upload.Store.Type != "fs" {
		c.String(http.StatusNotImplemented, "store type not supported")
		return
	}
	_ = app.App.DBIo.UpdateUploadSession(req.UploadID, map[string]any{
		"status": "aborted",
	})
	_ = os.RemoveAll(uploadDir(req.UploadID))
	c.String(http.StatusOK, "ok")
}

func downloadFile(c *gin.Context) {
	host := c.Query("host")
	path := c.Query("path")
	sshUserQuery := c.Query("user")
	if host == "" || path == "" {
		c.String(http.StatusBadRequest, "host and path required")
		return
	}
	if err := validateRemotePath(path); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	v, ok := c.Get("auth_user")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	authUser := v.(model.User)
	if authUser.Username == nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}

	server, sshUsers, err := app.App.Sshd.SshdIO.GetSSHUsersByHostLive(host)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	sshUser, err := selectSSHUser(sshUsers, sshUserQuery, "")
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if err := app.App.Sshd.SshdIO.CheckPermission(fmt.Sprintf("%s@%s:%s", sshUser.UserName, host, path), authUser, model.Download); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	_, upstream, err := sshd.NewSSHClient(*authUser.Username, *server, sshUser)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	defer upstream.Close()

	sftpClient, err := sftp.NewClient(upstream)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(path)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer remoteFile.Close()

	info, err := remoteFile.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	filename := filepath.Base(path)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Accept-Ranges", "bytes")

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		start, end, ok := parseRange(rangeHeader, info.Size())
		if !ok {
			c.String(http.StatusRequestedRangeNotSatisfiable, "invalid range")
			return
		}
		_, err = remoteFile.Seek(start, io.SeekStart)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		length := end - start + 1
		c.Header("Content-Length", fmt.Sprintf("%d", length))
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, info.Size()))
		c.Status(http.StatusPartialContent)
		_, _ = io.CopyN(c.Writer, remoteFile, length)
	} else {
		c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, remoteFile)
	}

	if app.App.DBIo != nil {
		action := "download"
		from := fmt.Sprintf("%s:%s", host, path)
		to := "browser"
		client := c.ClientIP()
		_ = app.App.DBIo.AddScpRecord(&model.AddScpRecordRequest{
			Action: &action,
			From:   &from,
			To:     &to,
			User:   authUser.Username,
			Client: &client,
		})
	}
}

func uploadDir(uploadID string) string {
	return filepath.Join(app.App.Config.Upload.Store.FSPath, uploadID)
}

func chunkPath(uploadID string, index int) string {
	return filepath.Join(uploadDir(uploadID), fmt.Sprintf("%06d.part", index))
}

func missingChunks(uploadID string, total int) []int {
	var missing []int
	for i := 0; i < total; i++ {
		if _, err := os.Stat(chunkPath(uploadID, i)); err != nil {
			missing = append(missing, i)
		}
	}
	return missing
}

func parseRange(h string, size int64) (int64, int64, bool) {
	if !strings.HasPrefix(h, "bytes=") {
		return 0, 0, false
	}
	raw := strings.TrimPrefix(h, "bytes=")
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	var end int64
	if parts[1] == "" {
		end = size - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	if start > end || end >= size {
		return 0, 0, false
	}
	return start, end, true
}

func valueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func validateRemotePath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must be absolute")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path")
	}
	return nil
}

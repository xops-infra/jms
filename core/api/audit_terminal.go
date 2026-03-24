package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
)

var terminalAuditNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.log$`)

// TerminalAuditFile describes one session log file under withVideo.dir.
type TerminalAuditFile struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
	Host    string `json:"host,omitempty"`
	User    string `json:"user,omitempty"`
}

// TerminalAuditListResponse lists terminal session recordings.
type TerminalAuditListResponse struct {
	Enabled bool                `json:"enabled"`
	Dir     string              `json:"dir,omitempty"`
	Files   []TerminalAuditFile `json:"files"`
	Total   int                 `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
	HasMore bool                `json:"has_more"`
}

func parseTerminalAuditName(name string) (host, user string) {
	base := strings.TrimSuffix(name, ".log")
	parts := strings.SplitN(base, "_", 4)
	if len(parts) != 4 {
		return "", ""
	}
	return parts[2], parts[3]
}

// @Summary listTerminalAuditLogs
// @Description 列出终端会话审计日志文件（withVideo 目录下），需管理员
// @Tags audit
// @Accept json
// @Produce json
// @Param limit query int false "max files, default 200"
// @Success 200 {object} TerminalAuditListResponse
// @Router /api/v1/audit/terminal [get]
func listTerminalAuditLogs(c *gin.Context) {
	resp := TerminalAuditListResponse{Enabled: false, Files: []TerminalAuditFile{}, Total: 0, Limit: 50, Offset: 0, HasMore: false}
	if !app.App.Config.WithVideo.Enable {
		c.JSON(http.StatusOK, resp)
		return
	}
	dir := strings.TrimSpace(app.App.Config.WithVideo.Dir)
	if dir == "" {
		c.JSON(http.StatusOK, resp)
		return
	}
	resp.Enabled = true
	resp.Dir = dir

	limit := 50
	if q := strings.TrimSpace(c.Query("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
			if limit > 200 {
				limit = 200
			}
		}
	}
	offset := 0
	if q := strings.TrimSpace(c.Query("offset")); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			offset = n
		}
	}
	resp.Limit = limit
	resp.Offset = offset

	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	all := make([]TerminalAuditFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		if !terminalAuditNameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		host, user := parseTerminalAuditName(e.Name())
		all = append(all, TerminalAuditFile{
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
			Host:    host,
			User:    user,
		})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ModTime > all[j].ModTime
	})
	resp.Total = len(all)
	if offset >= len(all) {
		resp.Files = []TerminalAuditFile{}
		resp.HasMore = false
		c.JSON(http.StatusOK, resp)
		return
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	resp.Files = all[offset:end]
	resp.HasMore = end < len(all)
	c.JSON(http.StatusOK, resp)
}

// @Summary getTerminalAuditLog
// @Description 下载终端会话审计日志原始内容（用于回放）
// @Tags audit
// @Produce octet-stream
// @Param name path string true "file name"
// @Success 200 {string} binary
// @Router /api/v1/audit/terminal/{name} [get]
func getTerminalAuditLog(c *gin.Context) {
	if !app.App.Config.WithVideo.Enable {
		c.String(http.StatusNotFound, "terminal audit disabled")
		return
	}
	dir := strings.TrimSpace(app.App.Config.WithVideo.Dir)
	if dir == "" {
		c.String(http.StatusNotFound, "audit dir not configured")
		return
	}
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	name := filepath.Base(c.Param("name"))
	if !terminalAuditNameRe.MatchString(name) {
		c.String(http.StatusBadRequest, "invalid file name")
		return
	}
	full := filepath.Join(dirAbs, name)
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	rel, err := filepath.Rel(dirAbs, fullAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		c.String(http.StatusBadRequest, "invalid path")
		return
	}
	st, err := os.Stat(fullAbs)
	if err != nil || st.IsDir() {
		c.String(http.StatusNotFound, "not found")
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "inline; filename=\""+name+"\"")
	c.Header("Content-Length", strconv.FormatInt(st.Size(), 10))
	f, err := os.Open(fullAbs)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()
	_, _ = io.Copy(c.Writer, f)
}

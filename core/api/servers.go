package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

type serverListItem struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Host    string      `json:"host"`
	User    string      `json:"user,omitempty"`
	Profile string      `json:"profile,omitempty"`
	Status  string      `json:"status,omitempty"`
	Tags    interface{} `json:"tags,omitempty"`
	Allowed bool        `json:"allowed"`
}

// listServers returns servers the user can connect to.
func listServers(c *gin.Context) {
	v, ok := c.Get("auth_user")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	authUser, ok := v.(model.User)
	if !ok || authUser.Username == nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}

	servers, err := app.App.DBIo.LoadServer()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	matchPolicies := app.App.Sshd.SshdIO.GetUserPolicys(*authUser.Username)
	items := make([]serverListItem, 0, len(servers))
	q := strings.TrimSpace(c.Query("q"))

	for _, server := range servers {
		allowed := app.App.Sshd.SshdIO.MatchPolicy(authUser, model.Connect, server, matchPolicies, false)
		if q != "" {
			needle := strings.ToLower(q)
			if !strings.Contains(strings.ToLower(server.Host), needle) &&
				!strings.Contains(strings.ToLower(server.Name), needle) {
				continue
			}
		}
		items = append(items, serverListItem{
			ID:      server.ID,
			Name:    server.Name,
			Host:    server.Host,
			User:    server.User,
			Profile: server.Profile,
			Status:  string(server.Status),
			Tags:    server.Tags,
			Allowed: allowed,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

type sshUserItem struct {
	User     string `json:"user"`
	KeyName  string `json:"key_name,omitempty"`
	AuthType string `json:"auth_type"`
	Source   string `json:"source,omitempty"`
}

// listServerSSHUsers returns ssh user options for the given host (permission required).
func listServerSSHUsers(c *gin.Context) {
	v, ok := c.Get("auth_user")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	authUser, ok := v.(model.User)
	if !ok || authUser.Username == nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}

	host := strings.TrimSpace(c.Param("host"))
	if host == "" {
		c.String(http.StatusBadRequest, "host required")
		return
	}

	server, sshUsers, fallbackUsers, err := app.App.Sshd.SshdIO.GetSSHUsersByHostResolvedLive(host)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	matchPolicies := app.App.Sshd.SshdIO.GetUserPolicys(*authUser.Username)
	if !app.App.Sshd.SshdIO.MatchPolicy(authUser, model.Connect, *server, matchPolicies, false) {
		c.String(http.StatusForbidden, "no permission")
		return
	}

	items := make([]sshUserItem, 0, len(sshUsers))
	fallbackIndex := make(map[string]struct{}, len(fallbackUsers))
	for _, u := range fallbackUsers {
		fallbackIndex[u.UserName+"|"+u.KeyName+"|"+u.Password] = struct{}{}
	}
	for _, u := range sshUsers {
		authType := "key"
		source := "managed_key"
		if u.Password != "" {
			authType = "password"
			source = "password"
		}
		if _, ok := fallbackIndex[u.UserName+"|"+u.KeyName+"|"+u.Password]; ok {
			source = "profile_fallback"
		}
		items = append(items, sshUserItem{
			User:     u.UserName,
			KeyName:  u.KeyName,
			AuthType: authType,
			Source:   source,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

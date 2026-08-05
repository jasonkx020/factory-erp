package mqtt

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"erp/internal/config"
	"erp/internal/security"
)

// AuthHandler NanoMQ HTTP Authentication / ACL callbacks (permit_all).
type AuthHandler struct {
	Cfg       *config.Config
	roleCache sync.Map // username -> []string roles
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{Cfg: cfg}
}

func (h *AuthHandler) Register(engine *gin.Engine) {
	engine.POST("/api/v1/mqtt/auth", h.handleAuth)
	engine.POST("/api/v1/mqtt/superuser", h.handleSuperuser)
	engine.POST("/api/v1/mqtt/acl", h.handleACL)
}

type mqttConnectAuthRequest struct {
	Username string
	Password string
	ClientID string
}

type nanomqACLRequest struct {
	Username string
	ClientID string
	Access   string
	Topic    string
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func isNanomqFormRequest(c *gin.Context) bool {
	ct := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
	return strings.Contains(ct, "application/x-www-form-urlencoded") ||
		strings.Contains(ct, "application/form-urlencoded")
}

func parseNanomqFormAuth(c *gin.Context) (mqttConnectAuthRequest, bool) {
	if err := c.Request.ParseForm(); err != nil {
		return mqttConnectAuthRequest{}, false
	}
	return mqttConnectAuthRequest{
		Username: strings.TrimSpace(c.PostForm("username")),
		Password: strings.TrimSpace(c.PostForm("password")),
		ClientID: strings.TrimSpace(firstNonEmpty(c.PostForm("clientid"), c.PostForm("client_id"))),
	}, true
}

func parseNanomqFormACL(c *gin.Context) (nanomqACLRequest, bool) {
	if err := c.Request.ParseForm(); err != nil {
		return nanomqACLRequest{}, false
	}
	return nanomqACLRequest{
		Username: strings.TrimSpace(c.PostForm("username")),
		ClientID: strings.TrimSpace(firstNonEmpty(c.PostForm("clientid"), c.PostForm("client_id"))),
		Access:   strings.TrimSpace(c.PostForm("access")),
		Topic:    strings.TrimSpace(c.PostForm("topic")),
	}, true
}

func (h *AuthHandler) mqttEnabled() bool {
	return h.Cfg != nil && h.Cfg.Mqtt.Enabled && h.Cfg.Mqtt.HTTPAuthEnabled
}

func (h *AuthHandler) isHubCredential(username, password string) bool {
	if h.Cfg == nil {
		return false
	}
	expectedUser := strings.TrimSpace(h.Cfg.Mqtt.ServerUsername)
	expectedPass := strings.TrimSpace(h.Cfg.Mqtt.ServerPassword)
	if expectedPass == "" {
		return false
	}
	return strings.TrimSpace(username) == expectedUser && strings.TrimSpace(password) == expectedPass
}

func (h *AuthHandler) isHubUsername(username string) bool {
	if h.Cfg == nil {
		return false
	}
	return strings.TrimSpace(username) == strings.TrimSpace(h.Cfg.Mqtt.ServerUsername)
}

func (h *AuthHandler) handleAuth(c *gin.Context) {
	if isNanomqFormRequest(c) {
		h.handleNanomqAuth(c)
		return
	}
	h.handleEMQXAuth(c)
}

func (h *AuthHandler) handleNanomqAuth(c *gin.Context) {
	if !h.mqttEnabled() {
		c.Status(http.StatusForbidden)
		return
	}
	req, ok := parseNanomqFormAuth(c)
	if !ok {
		c.Status(http.StatusForbidden)
		return
	}
	if allow, _ := h.authorizeConnect(req); allow {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusForbidden)
}

func (h *AuthHandler) handleSuperuser(c *gin.Context) {
	if !h.mqttEnabled() {
		c.Status(http.StatusForbidden)
		return
	}
	req, ok := parseNanomqFormAuth(c)
	if !ok {
		c.Status(http.StatusForbidden)
		return
	}
	if h.isHubCredential(req.Username, req.Password) {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusForbidden)
}

func (h *AuthHandler) handleACL(c *gin.Context) {
	if !h.mqttEnabled() {
		c.Status(http.StatusForbidden)
		return
	}
	req, ok := parseNanomqFormACL(c)
	if !ok || req.Topic == "" {
		c.Status(http.StatusForbidden)
		return
	}
	if h.isHubUsername(req.Username) {
		c.Status(http.StatusOK)
		return
	}
	if h.authorizeUserACL(req) {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusForbidden)
}

type emqxAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ClientID string `json:"clientid"`
}

type emqxAuthResponse struct {
	Result      string `json:"result"`
	IsSuperuser bool   `json:"is_superuser"`
}

func (h *AuthHandler) handleEMQXAuth(c *gin.Context) {
	if !h.mqttEnabled() {
		c.JSON(http.StatusOK, emqxAuthResponse{Result: "deny"})
		return
	}
	var req emqxAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, emqxAuthResponse{Result: "deny"})
		return
	}
	connectReq := mqttConnectAuthRequest{Username: req.Username, Password: req.Password, ClientID: req.ClientID}
	if allow, isHub := h.authorizeConnect(connectReq); allow {
		c.JSON(http.StatusOK, emqxAuthResponse{Result: "allow", IsSuperuser: isHub})
		return
	}
	c.JSON(http.StatusOK, emqxAuthResponse{Result: "deny"})
}

func (h *AuthHandler) authorizeConnect(req mqttConnectAuthRequest) (allow bool, isHub bool) {
	if !h.mqttEnabled() {
		return false, false
	}
	if h.isHubCredential(req.Username, req.Password) {
		return true, true
	}
	return h.validateUserConnect(req), false
}

func (h *AuthHandler) validateUserConnect(req mqttConnectAuthRequest) bool {
	password := strings.TrimSpace(req.Password)
	if password == "" || h.Cfg == nil {
		return false
	}
	claims, err := security.ParseMqttToken(h.Cfg.JWT.Secret, password)
	if err != nil {
		return false
	}
	if claims.Tenant != "" && claims.Tenant != Tenant(h.Cfg) {
		return false
	}
	if strings.TrimSpace(req.Username) == "" || req.Username != claims.Username {
		return false
	}
	if claims.ClientID != "" && strings.TrimSpace(req.ClientID) != claims.ClientID {
		return false
	}
	roles := append([]string{}, claims.Roles...)
	h.roleCache.Store(req.Username, roles)
	return true
}

func (h *AuthHandler) authorizeUserACL(req nanomqACLRequest) bool {
	access := strings.TrimSpace(req.Access)
	isSubscribe := access == "1" || strings.EqualFold(access, "subscribe") || strings.EqualFold(access, "sub")
	isPublish := access == "2" || strings.EqualFold(access, "publish") || strings.EqualFold(access, "pub")
	if isPublish || !isSubscribe {
		return false
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || !strings.HasPrefix(username, "u") {
		return false
	}
	uidStr := strings.TrimPrefix(username, "u")
	userID, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || userID <= 0 {
		return false
	}
	roles := []string{}
	if v, ok := h.roleCache.Load(username); ok {
		if rr, ok2 := v.([]string); ok2 {
			roles = rr
		}
	}
	return TopicAllowedForUser(Tenant(h.Cfg), userID, roles, req.Topic)
}

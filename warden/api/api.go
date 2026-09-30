package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gaucho-racing/warden/warden/bridge"
	"github.com/gaucho-racing/warden/warden/config"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
	"github.com/gaucho-racing/warden/warden/pkg/sentinel"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// MinecraftAdminGroup gates Warden's entire server-administration surface:
// bindings, the player roster, and the audit log. It is the only privileged
// group Warden knows about — org-wide Admins deliberately grants nothing
// here, because running the game server is not the same job.
const MinecraftAdminGroup = "MinecraftAdmins"

func Run() {
	router := InitializeRouter()
	InitializeRoutes(router)
	handler := withPluginSocket(router, http.HandlerFunc(serveBridge))
	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		logger.SugarLogger.Fatalf("Failed to start server: %v", err)
	}
}

const pluginSocketPath = "/plugin/bridge"

// withPluginSocket serves the plugin's WebSocket outside gin. coder/websocket
// calls gin's WriteHeaderNow before hijacking (a workaround for older gin),
// and gin 1.11 refuses to hijack a response it considers written, so the
// upgrade cannot go through a gin handler at all. The token check mirrors
// AuthChecker's plugin branch.
func withPluginSocket(router http.Handler, socket http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pluginSocketPath {
			router.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		bearer, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(bearer), []byte(config.PluginToken)) != 1 {
			http.Error(w, `{"error":"invalid plugin token"}`, http.StatusUnauthorized)
			return
		}
		socket.ServeHTTP(w, r)
	})
}

// serveBridge answers 503 when the bridge is off, which the plugin treats as
// "retry later".
func serveBridge(w http.ResponseWriter, r *http.Request) {
	b := bridge.Current()
	if b == nil {
		http.Error(w, `{"error":"discord bridge is disabled"}`, http.StatusServiceUnavailable)
		return
	}
	b.ServePlugin(w, r)
}

func InitializeRouter() *gin.Engine {
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: false,
	}))
	r.Use(AuthChecker())
	r.Use(UnauthorizedPanicHandler())
	return r
}

func InitializeRoutes(router *gin.Engine) {
	router.GET("/ping", Ping)

	router.POST("/auth/login", LoginWithSentinel)
	router.GET("/auth/session", GetSession)
	router.POST("/auth/refresh", RefreshSession)
	router.POST("/auth/logout", Logout)
	router.GET("/users/@me", GetCurrentUser)
	router.GET("/groups", ListBindableGroups)

	router.GET("/accounts", ListAccounts)
	router.GET("/accounts/@me", GetMyAccount)
	router.GET("/accounts/:uuid", GetAccount)
	router.DELETE("/accounts/:uuid", DeleteAccount)

	router.GET("/stats/@me", GetMyStats)
	router.GET("/stats/:uuid", GetPlayerStats)

	router.GET("/link/:token", GetLinkToken)
	router.POST("/link/:token", ConsumeLinkToken)

	router.GET("/bindings", ListBindings)
	router.POST("/bindings", CreateBinding)
	router.GET("/bindings/:id", GetBinding)
	router.PUT("/bindings/:id", UpdateBinding)
	router.DELETE("/bindings/:id", DeleteBinding)

	router.GET("/audit-logs", ListAuditLogs)

	// Plugin realm. Authenticated by PLUGIN_TOKEN, not by Sentinel — the
	// game server is a lower-trust client and never holds a Sentinel
	// credential. Read-only with respect to Sentinel: nothing under here
	// can write group membership.
	router.POST("/plugin/link-tokens", IssueLinkToken)
	router.GET("/plugin/players/:uuid", GetPlayerPermissions)
	router.POST("/plugin/players/:uuid/seen", MarkPlayerSeen)
	router.POST("/plugin/players/:uuid/stats", ReportPlayerStats)
	router.GET("/plugin/sync", SyncAllPlayers)
}

// AuthChecker resolves one of two independent credentials depending on the
// route: a Sentinel-issued user JWT for the web portal, or the static
// PLUGIN_TOKEN for the game server. They never overlap — a plugin token is
// worthless on a portal route and vice versa.
func AuthChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		bearer := bearerToken(c)

		if strings.HasPrefix(path, "/plugin/") {
			if subtle.ConstantTimeCompare([]byte(bearer), []byte(config.PluginToken)) != 1 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid plugin token"})
				return
			}
			c.Set("Auth-Plugin", true)
			c.Next()
			return
		}

		if authRouteSkipsTokenValidation(path) {
			c.Next()
			return
		}

		if bearer != "" {
			claims, err := sentinel.ValidateToken(bearer)
			if err != nil {
				logger.SugarLogger.Errorln("Failed to validate token: " + err.Error())
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			setAuthContext(c, bearer, claims)
			if !RequestTokenHasAudience(c, config.SentinelClientID) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token is not intended for Warden"})
				return
			}
			logger.SugarLogger.Infof("Decoded token: entity=%s audience=%s scope=%s", GetRequestTokenEntityID(c), GetRequestTokenAudience(c), GetRequestTokenScopes(c))
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

func authRouteSkipsTokenValidation(path string) bool {
	return path == "/ping" ||
		path == "/auth/login" ||
		path == "/auth/refresh" ||
		path == "/auth/logout"
}

func UnauthorizedPanicHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if err == "Unauthorized" {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "you are not authorized to access this resource"})
					return
				}
				logger.SugarLogger.Errorf("Unexpected panic: %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprint(err)})
			}
		}()
		c.Next()
	}
}

// nowUTC centralizes the clock so link-token expiry comparisons in the API
// layer all read from the same place.
func nowUTC() time.Time {
	return time.Now()
}

func Require(c *gin.Context, condition bool) {
	if !condition {
		panic("Unauthorized")
	}
}

func Any(conditions ...bool) bool {
	for _, condition := range conditions {
		if condition {
			return true
		}
	}
	return false
}

func All(conditions ...bool) bool {
	for _, condition := range conditions {
		if !condition {
			return false
		}
	}
	return true
}

func RequestTokenExists(c *gin.Context) bool {
	_, exists := c.Get("Auth-Token")
	return exists
}

func RequestIsPlugin(c *gin.Context) bool {
	value, exists := c.Get("Auth-Plugin")
	if !exists {
		return false
	}
	isPlugin, ok := value.(bool)
	return ok && isPlugin
}

func RequestTokenHasAudience(c *gin.Context, audience string) bool {
	return GetRequestTokenAudience(c) == audience
}

func RequestTokenHasEntityID(c *gin.Context, entityID string) bool {
	return GetRequestTokenEntityID(c) == entityID
}

func RequestTokenHasGroupName(c *gin.Context, groupName string) bool {
	for _, tokenGroup := range GetRequestTokenGroupNames(c) {
		if tokenGroup == groupName {
			return true
		}
	}
	return false
}

func RequestUserIsMinecraftAdmin(c *gin.Context) bool {
	return RequestTokenHasGroupName(c, MinecraftAdminGroup)
}

func RequestTokenCanManageBindings(c *gin.Context) bool {
	return RequestUserIsMinecraftAdmin(c)
}

func GetRequestToken(c *gin.Context) string {
	token, _ := c.Get("Auth-Token")
	return contextString(token)
}

func GetRequestTokenScopes(c *gin.Context) string {
	scopes, _ := c.Get("Auth-Scope")
	return contextString(scopes)
}

func GetRequestTokenAudience(c *gin.Context) string {
	audience, _ := c.Get("Auth-Audience")
	return contextString(audience)
}

func GetRequestTokenClaims(c *gin.Context) map[string]interface{} {
	claims, exists := c.Get("Auth-Claims")
	if !exists {
		return nil
	}
	value, ok := claims.(map[string]interface{})
	if !ok {
		return nil
	}
	return value
}

func GetRequestTokenEntityID(c *gin.Context) string {
	entityID, _ := c.Get("Auth-EntityID")
	return contextString(entityID)
}

func GetRequestTokenUserID(c *gin.Context) string {
	return claimString(GetRequestTokenClaims(c), "user_id")
}

func GetRequestTokenGroupNames(c *gin.Context) []string {
	return claimStringSlice(GetRequestTokenClaims(c), "groups")
}

func setAuthContext(c *gin.Context, token string, claims map[string]interface{}) {
	c.Set("Auth-Token", token)
	c.Set("Auth-Claims", claims)
	c.Set("Auth-EntityID", claimString(claims, "sub"))
	c.Set("Auth-Scope", claimString(claims, "scope"))
	c.Set("Auth-UserID", claimString(claims, "user_id"))
	audiences := claimStringSlice(claims, "aud")
	if len(audiences) > 0 {
		c.Set("Auth-Audience", audiences[0])
	}
}

func contextString(value interface{}) string {
	if value == nil {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

func claimString(claims map[string]interface{}, key string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return value
}

func claimStringSlice(claims map[string]interface{}, key string) []string {
	if claims == nil {
		return []string{}
	}
	switch value := claims[key].(type) {
	case []string:
		return value
	case []interface{}:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	case string:
		if value == "" {
			return []string{}
		}
		return []string{value}
	default:
		return []string{}
	}
}

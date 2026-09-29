package sentinel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gaucho-racing/warden/warden/config"
)

type Error struct {
	Code    int
	Message string `json:"error"`
}

func (e Error) Error() string {
	if e.Code == 0 {
		return e.Message
	}
	return fmt.Sprintf("sentinel error: [%d] %s", e.Code, e.Message)
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type User struct {
	ID          string   `json:"id"`
	EntityID    string   `json:"entity_id"`
	Username    string   `json:"username"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Email       string   `json:"email"`
	AvatarURL   string   `json:"avatar_url"`
	InitialRole string   `json:"initial_role"`
	Groups      []string `json:"groups"`
	UpdatedAt   string   `json:"updated_at"`
	CreatedAt   string   `json:"created_at"`
}

type IdentityApplicationSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ClientID string `json:"client_id"`
	IconURL  string `json:"icon_url"`
}

type IdentitySummary struct {
	ID          string                      `json:"id"`
	Type        string                      `json:"type"`
	Name        string                      `json:"name"`
	Username    string                      `json:"username,omitempty"`
	AvatarURL   string                      `json:"avatar_url,omitempty"`
	Application *IdentityApplicationSummary `json:"application,omitempty"`
}

type Group struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	AllowedSources []string `json:"allowed_sources"`
	CreatedBy      string   `json:"created_by"`
	MemberCount    int64    `json:"member_count"`
	OwnerCount     int64    `json:"owner_count"`
	PendingCount   int64    `json:"pending_count"`
	UpdatedAt      string   `json:"updated_at"`
	CreatedAt      string   `json:"created_at"`
}

// Application is the Sentinel app record. Warden only ever looks up its
// own, to find which groups have been linked to it.
type Application struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ClientID string `json:"client_id"`
	IconURL  string `json:"icon_url"`
}

// ApplicationGroup is a group linked to an application. Required marks the
// ones Sentinel enforces at authorize time.
type ApplicationGroup struct {
	Group
	Required bool `json:"required"`
}

type GroupMember struct {
	GroupID  string `json:"group_id"`
	EntityID string `json:"entity_id"`
	Source   string `json:"source"`
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func ExchangeAuthorizationCode(code string, redirectURI string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	return exchangeToken(form)
}

func RefreshToken(refreshToken string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return exchangeToken(form)
}

// GetCurrentUser reads a user with the caller's own access token, so it only
// ever returns what that user is allowed to see.
func GetCurrentUser(ctx context.Context, accessToken string, userID string) (User, error) {
	if strings.TrimSpace(userID) == "" {
		return User{}, fmt.Errorf("user id is required")
	}
	var user User
	err := get(ctx, accessToken, "/api/users/"+url.PathEscape(userID), &user)
	return user, err
}

// GetGroups lists every Sentinel group visible to the token. The binding
// editor uses the caller's token; the reconcile sweep uses the service
// account.
func GetGroups(ctx context.Context, accessToken string) ([]Group, error) {
	groups := []Group{}
	err := get(ctx, accessToken, "/api/groups", &groups)
	return groups, err
}

// GetEntityGroups returns the groups an entity belongs to. This is the fast
// path used when a single player joins: one call, no reverse index. Requires
// groups:read, which Warden's service account holds.
func GetEntityGroups(ctx context.Context, accessToken string, entityID string) ([]Group, error) {
	if strings.TrimSpace(entityID) == "" {
		return nil, fmt.Errorf("entity id is required")
	}
	groups := []Group{}
	err := get(ctx, accessToken, "/api/core/entity/"+url.PathEscape(entityID)+"/groups", &groups)
	return groups, err
}

// GetGroupMembers returns every member of a group. The reconcile sweep uses
// this instead of GetEntityGroups per player: one call per bound group beats
// one call per linked account once there are more players than bindings.
func GetGroupMembers(ctx context.Context, accessToken string, groupID string) ([]GroupMember, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, fmt.Errorf("group id is required")
	}
	members := []GroupMember{}
	err := get(ctx, accessToken, "/api/groups/"+url.PathEscape(groupID)+"/members", &members)
	return members, err
}

// GetApplicationByClientID resolves a client_id to its application record.
// Needs the applications:read scope.
func GetApplicationByClientID(ctx context.Context, accessToken string, clientID string) (Application, error) {
	if strings.TrimSpace(clientID) == "" {
		return Application{}, fmt.Errorf("client id is required")
	}
	var app Application
	err := get(ctx, accessToken, "/api/applications/client/"+url.PathEscape(clientID), &app)
	return app, err
}

// GetApplicationGroups lists the groups linked to an application. This is
// what scopes Warden's binding editor: only groups an admin has deliberately
// attached to the Warden app in Sentinel can be bound to a Minecraft role.
func GetApplicationGroups(ctx context.Context, accessToken string, applicationID string) ([]ApplicationGroup, error) {
	if strings.TrimSpace(applicationID) == "" {
		return nil, fmt.Errorf("application id is required")
	}
	groups := []ApplicationGroup{}
	err := get(ctx, accessToken, "/api/applications/"+url.PathEscape(applicationID)+"/groups", &groups)
	return groups, err
}

// ResolveIdentities batch-resolves entity IDs to display summaries so list
// views can show real names without a request per row.
func ResolveIdentities(ctx context.Context, accessToken string, entityIDs []string) ([]IdentitySummary, error) {
	if len(entityIDs) == 0 {
		return []IdentitySummary{}, nil
	}
	body, err := json.Marshal(map[string][]string{"ids": entityIDs})
	if err != nil {
		return nil, fmt.Errorf("encode identity summary request: %w", err)
	}
	summaries := []IdentitySummary{}
	if err := do(ctx, http.MethodPost, accessToken, "/api/entities/resolve", body, &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

func get(ctx context.Context, accessToken string, path string, result any) error {
	return do(ctx, http.MethodGet, accessToken, path, nil, result)
}

func do(ctx context.Context, method string, accessToken string, path string, body []byte, result any) error {
	if strings.TrimSpace(config.SentinelURL) == "" {
		return fmt.Errorf("SENTINEL_URL is not configured")
	}
	if strings.TrimSpace(accessToken) == "" {
		return fmt.Errorf("access token is required")
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(config.SentinelURL, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var sentinelErr Error
		if json.Unmarshal(respBody, &sentinelErr) == nil && sentinelErr.Message != "" {
			sentinelErr.Code = resp.StatusCode
			return sentinelErr
		}
		return Error{Code: resp.StatusCode, Message: strings.TrimSpace(string(respBody))}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(respBody, result)
}

func exchangeToken(form url.Values) (TokenResponse, error) {
	if strings.TrimSpace(config.SentinelURL) == "" {
		return TokenResponse{}, fmt.Errorf("SENTINEL_URL is not configured")
	}
	if strings.TrimSpace(config.SentinelClientID) == "" {
		return TokenResponse{}, fmt.Errorf("SENTINEL_CLIENT_ID is not configured")
	}
	if strings.TrimSpace(config.SentinelClientSecret) == "" {
		return TokenResponse{}, fmt.Errorf("SENTINEL_CLIENT_SECRET is not configured")
	}
	form.Set("client_id", config.SentinelClientID)
	form.Set("client_secret", config.SentinelClientSecret)

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(config.SentinelURL, "/")+"/api/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return TokenResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		var sentinelErr Error
		if err := json.Unmarshal(respBody, &sentinelErr); err != nil {
			return TokenResponse{}, err
		}
		sentinelErr.Code = resp.StatusCode
		return TokenResponse{}, sentinelErr
	}

	var token TokenResponse
	if err := json.Unmarshal(respBody, &token); err != nil {
		return TokenResponse{}, err
	}
	if token.AccessToken == "" {
		return TokenResponse{}, fmt.Errorf("sentinel token response did not include access token")
	}
	return token, nil
}

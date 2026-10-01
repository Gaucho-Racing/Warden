// Package depot is a minimal client for the Depot object store, covering
// only the presigned upload flow Warden needs for server backups.
//
// Warden authenticates with the same Sentinel service-account token it uses
// everywhere else. Depot authorizes writes by the token's audience — the
// client_id of the Warden application — against a grant on the bucket, so
// nothing Depot-specific has to be provisioned beyond that grant.
package depot

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
	return fmt.Sprintf("depot error: [%d] %s", e.Code, e.Message)
}

// File mirrors the subset of Depot's file record Warden reads back.
type File struct {
	ID          string `json:"id"`
	BucketName  string `json:"bucket_name"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Status      string `json:"status"`
}

// Upload is a reserved file plus the URL its bytes go to. The URL is
// presigned against the storage backend directly, so whoever holds it
// uploads without a Depot credential and without Depot proxying the body.
type Upload struct {
	File      File      `json:"file"`
	URL       string    `json:"upload_url"`
	Method    string    `json:"method"`
	ExpiresAt time.Time `json:"expires_at"`
}

type InitiateRequest struct {
	OriginalName string            `json:"original_name"`
	Path         string            `json:"path"`
	ContentType  string            `json:"content_type"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// Depot answers slowly only when object storage does; these calls are
// metadata-only, so a short timeout is right. The upload itself is not made
// by this client.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// Enabled reports whether Warden holds the credential Depot needs. The
// origin and bucket are compiled in, so the service account is the only
// thing that can be missing.
func Enabled() bool {
	return config.SentinelSAToken != ""
}

// InitiateUpload reserves a file in the bucket and returns a presigned PUT
// URL for its contents. The file stays PENDING, and invisible to normal
// listings, until CompleteUpload confirms the object landed.
func InitiateUpload(ctx context.Context, bucket string, req InitiateRequest) (Upload, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Upload{}, fmt.Errorf("encode upload request: %w", err)
	}
	var upload Upload
	path := "/api/buckets/" + url.PathEscape(bucket) + "/uploads"
	if err := do(ctx, http.MethodPost, path, body, &upload); err != nil {
		return Upload{}, err
	}
	return upload, nil
}

// CompleteUpload promotes the file to ACTIVE. Depot stats the object to
// learn its real size, so the size it returns is what storage actually
// holds rather than what the uploader claimed.
func CompleteUpload(ctx context.Context, bucket string, fileID string) (File, error) {
	var file File
	path := "/api/buckets/" + url.PathEscape(bucket) + "/uploads/" + url.PathEscape(fileID) + "/complete"
	if err := do(ctx, http.MethodPost, path, nil, &file); err != nil {
		return File{}, err
	}
	return file, nil
}

func do(ctx context.Context, method string, path string, body []byte, result any) error {
	if strings.TrimSpace(config.DepotURL) == "" {
		return fmt.Errorf("DEPOT_URL is not configured")
	}
	if strings.TrimSpace(config.SentinelSAToken) == "" {
		return fmt.Errorf("SENTINEL_SA_TOKEN is not configured")
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(config.DepotURL, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.SentinelSAToken)
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
		var depotErr Error
		if json.Unmarshal(respBody, &depotErr) == nil && depotErr.Message != "" {
			depotErr.Code = resp.StatusCode
			return depotErr
		}
		return Error{Code: resp.StatusCode, Message: strings.TrimSpace(string(respBody))}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(respBody, result)
}

// Download is a time-limited, credential-free URL to a stored file.
type Download struct {
	URL       string    `json:"url"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateDownloadURL mints a link that fetches the archive straight from
// Depot without a Sentinel session. The token is bearer authority over that
// one file for an hour, which is why the route that calls this is restricted
// to MinecraftAdmins — a world archive contains every player's inventory and
// every sign on the map.
//
// Depot builds the absolute URL from the request it sees, so the path is
// rejoined to the configured origin here rather than trusted blindly.
func CreateDownloadURL(ctx context.Context, bucket string, fileID string) (Download, error) {
	var download Download
	path := "/api/buckets/" + url.PathEscape(bucket) + "/files/" + url.PathEscape(fileID) + "/download-url"
	if err := do(ctx, http.MethodPost, path, nil, &download); err != nil {
		return Download{}, err
	}
	if download.Path != "" {
		download.URL = strings.TrimRight(config.DepotURL, "/") + download.Path
	}
	return download, nil
}

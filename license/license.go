package license

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/example/porn2oautobuyer/hwid"
)

const (
	DefaultServerURL     = "https://autobuy.cloudpub.ru"
	defaultClientVersion = "2.5.1"
	verificationInterval = time.Hour
)

type Client struct {
	ServerURL     string
	ClientVersion string

	configDir   string
	licenseFile string
	lastKeyFile string
	httpClient  *http.Client

	lastVerification time.Time
}

type licenseFileData struct {
	Key       string    `json:"key"`
	HWID      string    `json:"hwid"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Result struct {
	Valid   bool
	Message string
}

func NewClient() (*Client, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return &Client{
		ServerURL:     DefaultServerURL,
		ClientVersion: defaultClientVersion,
		configDir:     dir,
		licenseFile:   filepath.Join(dir, "license.json"),
		lastKeyFile:   filepath.Join(dir, "last_key.json"),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func configDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = home
		}
		return filepath.Join(base, "Porn2oAutoBuyer"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "Porn2oAutoBuyer"), nil
	default:
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".config")
		}
		return filepath.Join(base, "porn2oautobuyer"), nil
	}
}

func (c *Client) RunLicensingCheck() (Result, error) {
	if time.Since(c.lastVerification) < verificationInterval && !c.lastVerification.IsZero() {
		return Result{Valid: true, Message: "cached"}, nil
	}

	key := os.Getenv("LICENSE_KEY")
	if key == "" {
		var err error
		key, err = c.loadLastKey()
		if err != nil {
			return Result{}, err
		}
	}
	if key == "" {
		fmt.Print("Введите лицензионный ключ: ")
		if _, err := fmt.Scanln(&key); err != nil {
			return Result{}, fmt.Errorf("read license key: %w", err)
		}
	}

	hwidString, err := c.collectHWID()
	if err != nil {
		return Result{}, err
	}

	if err := c.validateWithServer(key, hwidString); err != nil {
		return Result{}, err
	}

	data := licenseFileData{
		Key:       key,
		HWID:      hwidString,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := c.saveLicense(data); err != nil {
		return Result{}, err
	}
	if err := c.saveLastKey(key); err != nil {
		return Result{}, err
	}

	c.lastVerification = time.Now()
	return Result{Valid: true, Message: "ok"}, nil
}

func (c *Client) collectHWID() (string, error) {
	info, err := hwid.Collect()
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(info.HardwareID()))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (c *Client) validateWithServer(key, hwid string) error {
	if key == "" {
		return errors.New("empty license key")
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.ServerURL, "/")+"/api/license/verify", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Porn2oAutoBuyer/"+c.ClientVersion)
	q := req.URL.Query()
	q.Set("key", key)
	q.Set("hwid", hwid)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("license server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("license rejected: %s", resp.Status)
	}

	return nil
}

func (c *Client) saveLicense(data licenseFileData) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.licenseFile, bytes, 0o600)
}

func (c *Client) saveLastKey(key string) error {
	payload := map[string]string{"key": key}
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.lastKeyFile, bytes, 0o600)
}

func (c *Client) loadLastKey() (string, error) {
	bytes, err := os.ReadFile(c.lastKeyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var payload map[string]string
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return "", err
	}
	return payload["key"], nil
}

func (c *Client) LoadLicense() (licenseFileData, error) {
	bytes, err := os.ReadFile(c.licenseFile)
	if err != nil {
		return licenseFileData{}, err
	}
	var data licenseFileData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return licenseFileData{}, err
	}
	return data, nil
}

package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type RetryConfig struct {
	Max    int
	Base   float64
	Jitter float64
}

type Product struct {
	ID   string
	Name string
	Link string
}

type Config struct {
	APIID         int
	APIHash       string
	Phone         string
	Bot           string
	Session       string
	ProductLink   string
	Verbose       bool
	PreemptiveQty bool
	StartInterval float64
	QtyPreDelay   float64
	Retries       RetryConfig
}

type Result struct {
	Config   Config
	Products map[string]Product
}

var floatPattern = regexp.MustCompile(`^[-+]?\d+(?:\.\d+)?$`)

func Load(path string) (Result, error) {
	var result Result
	result.Products = make(map[string]Product)

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := writeTemplate(path); err != nil {
				return result, fmt.Errorf("create config template: %w", err)
			}
			return result, fmt.Errorf("config file %s not found: template created", path)
		}
		return result, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	line := 0
	retries := RetryConfig{Max: 3, Base: 0.15, Jitter: 0.05}
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		parts := strings.SplitN(text, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch {
		case strings.HasPrefix(key, "PRODUCT_"):
			productID := strings.TrimPrefix(key, "PRODUCT_")
			name, link := parseProduct(value)
			if link == "" {
				continue
			}
			result.Products[productID] = Product{ID: productID, Name: name, Link: link}
		case strings.HasPrefix(key, "RETRIES_"):
			key = strings.TrimPrefix(key, "RETRIES_")
			switch strings.ToLower(key) {
			case "max":
				if v, err := strconv.Atoi(value); err == nil {
					retries.Max = v
				}
			case "base":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					retries.Base = v
				}
			case "jitter":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					retries.Jitter = v
				}
			}
		default:
			applyScalar(&result.Config, key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scan config: %w", err)
	}
	result.Config.Retries = retries

	return result, nil
}

func applyScalar(cfg *Config, key, value string) {
	lower := strings.ToLower(key)
	switch lower {
	case "api_id":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.APIID = v
		}
	case "api_hash":
		cfg.APIHash = value
	case "phone":
		cfg.Phone = value
	case "bot":
		cfg.Bot = value
	case "session":
		cfg.Session = value
	case "product_link":
		cfg.ProductLink = value
	case "verbose":
		cfg.Verbose = parseBool(value)
	case "preemptive_qty":
		cfg.PreemptiveQty = parseBool(value)
	case "start_interval":
		cfg.StartInterval = parseFloat(value)
	case "qty_pre_delay":
		cfg.QtyPreDelay = parseFloat(value)
	}
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func parseFloat(v string) float64 {
	if !floatPattern.MatchString(v) {
		return 0
	}
	value, _ := strconv.ParseFloat(v, 64)
	return value
}

func parseProduct(value string) (string, string) {
	if value == "" {
		return "", ""
	}
	parts := strings.SplitN(value, "|", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func writeTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	template := `# config.txt
API_ID=123456
API_HASH=your_api_hash
PHONE=+79990000000
BOT=@your_bot_username
SESSION=finsal_session
PRODUCT_LINK=c_xxx
VERBOSE=false
PREEMPTIVE_QTY=true
START_INTERVAL=1.5
QTY_PRE_DELAY=1.0
RETRIES_MAX=3
RETRIES_BASE=0.15
RETRIES_JITTER=0.05
`
	return os.WriteFile(path, []byte(template), 0o644)
}

package config

import (
	"errors"
	"os"
	"strings"
)

// Config is read from the environment. The `.env` file is a symlink to the
// workspace's shared ../files/.env (the same one ships-go reads), and this
// service deliberately uses the same PORT (default 3000) and origins as
// ships-go: the 3D site is served on the 2D site's ports, so the 2D and 3D
// games can't run at the same time. SHIPS3D_* variables override for the 3D
// game only.
type Config struct {
	Port           string
	AllowedOrigins []string
	CertPath       string
	KeyPath        string
	MongoURI       string
	MongoDatabase  string
}

func Load() (Config, error) {
	cfg := Config{
		Port:          envOr("SHIPS3D_PORT", envOr("PORT", "3000")),
		CertPath:      os.Getenv("SSL_CERT_PATH"),
		KeyPath:       os.Getenv("SSL_KEY_PATH"),
		MongoURI:      os.Getenv("MONGODB_URI"),
		MongoDatabase: envOr("MONGODB_DATABASE", "jaes"),
	}
	if cfg.MongoURI == "" {
		return cfg, errors.New("MONGODB_URI is not set")
	}
	// ALLOWED_ORIGINS is ships-go's list, which already covers the 3D site
	// since it runs on the same ports; SHIPS3D_ALLOWED_ORIGINS can add more
	// (e.g. a production domain of its own). Both are "|"-separated.
	for _, list := range []string{os.Getenv("ALLOWED_ORIGINS"), os.Getenv("SHIPS3D_ALLOWED_ORIGINS")} {
		for _, origin := range strings.Split(list, "|") {
			if origin = strings.TrimSpace(origin); origin != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
			}
		}
	}
	return cfg, nil
}

// TLS reports whether a certificate is configured. Without one the server
// listens on plain HTTP, which is what a TLS-terminating proxy (or a future
// mobile app's local webview bridge) would want.
func (c Config) TLS() bool {
	return c.CertPath != "" && c.KeyPath != ""
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

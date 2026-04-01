package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"sync"
)

var (
	emailMap   map[string]string
	emailMapMu sync.RWMutex
	hostPrefix string
)

func loadEmailMap(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read email map: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse email map: %w", err)
	}
	emailMapMu.Lock()
	emailMap = m
	emailMapMu.Unlock()
	log.Printf("loaded %d email→user mappings", len(m))
	return nil
}

func getEmail(email string) (string, bool) {
	emailMapMu.RLock()
	defer emailMapMu.RUnlock()
	user, ok := emailMap[email]
	return user, ok
}

func extractUserFromHost(host string) (string, bool) {
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(hostPrefix) + `-([a-zA-Z0-9_-]+)-`)
	matches := pattern.FindStringSubmatch(host)
	if len(matches) < 2 {
		return "", false
	}
	return matches[1], true
}

func authHandler(proxyURL string, r *http.Request) (string, error) {
	authURL := proxyURL + "/oauth2/auth"
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", r.Header.Get("Cookie"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return "", nil
	}

	return resp.Header.Get("X-Auth-Request-Email"), nil
}

func main() {
	oauthProxyURL := getEnv("OAUTH2_PROXY_URL", "http://ok8s-oauth2-proxy:4180")
	signInURL := getEnv("SIGNIN_URL", "https://oc-ingress-opencode/oauth2/sign_in")
	mapPath := getEnv("EMAIL_MAP_PATH", "/etc/email-map/email-map.json")
	hostPrefix = getEnv("HOST_PREFIX", "opencode")

	if err := loadEmailMap(mapPath); err != nil {
		log.Fatalf("failed to load email map: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		email, err := authHandler(oauthProxyURL, r)
		if err != nil {
			log.Printf("auth check error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if email == "" {
			http.Redirect(w, r, signInURL, http.StatusFound)
			return
		}

		expectedUser, ok := extractUserFromHost(r.Host)
		if !ok {
			http.Error(w, "Invalid hostname", http.StatusBadRequest)
			return
		}

		actualUser, found := getEmail(email)
		if !found {
			http.Error(w, fmt.Sprintf("Email %s not mapped to any user", email), http.StatusForbidden)
			return
		}

		if actualUser != expectedUser {
			log.Printf("access denied: email=%s mapped to user=%s, requested user=%s", email, actualUser, expectedUser)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		target := fmt.Sprintf("http://%s-user-%s:4096", hostPrefix, expectedUser)
		targetURL, _ := url.Parse(target)
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		r.Header.Set("X-Forwarded-Email", email)
		r.Header.Set("X-Forwarded-User", actualUser)

		proxy.ServeHTTP(w, r)
	})

	log.Printf("auth router listening on :8080")
	log.Printf("oauth2-proxy: %s", oauthProxyURL)
	log.Printf("email map: %s", mapPath)
	log.Printf("host prefix: %s", hostPrefix)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

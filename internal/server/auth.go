package server

import (
	"net/http"
	"time"
)

const tokenCookieName = "ht_token"

// authMiddleware checks for valid token in query param or cookie
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query param first
		queryToken := r.URL.Query().Get("token")
		if queryToken != "" {
			if queryToken == s.token {
				// Valid token in query param - set cookie and redirect without param
				http.SetCookie(w, &http.Cookie{
					Name:     tokenCookieName,
					Value:    s.token,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   int(24 * time.Hour / time.Second), // 24 hours
					SameSite: http.SameSiteLaxMode,
				})

				// Redirect to same path without token param
				newURL := *r.URL
				q := newURL.Query()
				q.Del("token")
				newURL.RawQuery = q.Encode()
				http.Redirect(w, r, newURL.String(), http.StatusFound)
				return
			}
			// Invalid token
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check cookie
		cookie, err := r.Cookie(tokenCookieName)
		if err != nil || cookie.Value != s.token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Valid cookie - proceed
		next.ServeHTTP(w, r)
	})
}

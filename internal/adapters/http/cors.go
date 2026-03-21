package httpadapter

import (
	"net/http"
	"strings"
)

// CORS returns middleware that adds Cross-Origin Resource Sharing headers for
// requests from explicitly allowed origins. Wildcard origins are not supported.
//
// Behaviour:
//   - No Origin header (non-browser SDK): request passes through unchanged.
//   - Origin matches an allowed origin: Access-Control-Allow-Origin and Vary
//     headers are added. For OPTIONS preflight requests the middleware also sets
//     Access-Control-Allow-Methods and Access-Control-Allow-Headers then calls
//     the next handler (which should return 200).
//   - Origin does not match: no CORS headers are added; the browser enforces
//     the block.
//
// If allowedOrigins is empty the middleware is a no-op.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
					if r.Method == http.MethodOptions {
						w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
						w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

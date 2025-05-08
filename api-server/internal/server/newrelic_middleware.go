package server

import (
	"net/http"
	
	"github.com/newrelic/go-agent/v3/newrelic"
)

// NewRelicMiddleware wraps handlers with New Relic transaction monitoring
func (s *Server) NewRelicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.nrApp == nil {
			// If New Relic isn't initialized, just call the next handler
			next.ServeHTTP(w, r)
			return
		}
		
		// Start a new transaction
		txn := s.nrApp.StartTransaction(r.URL.Path)
		defer txn.End()
		
		// Add the transaction to the request context
		r = newrelic.RequestWithTransactionContext(r, txn)
		
		// Use a wrapped response writer that reports to New Relic
		w = txn.SetWebResponse(w)
		
		// Call the next handler with the augmented request/response
		next.ServeHTTP(w, r)
	})
}
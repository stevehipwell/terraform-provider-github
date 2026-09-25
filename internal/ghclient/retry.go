package ghclient

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

// checkRetryNoRatelimit extends the default retry policy to ignore [http.StatusTooManyRequests] responses.
func checkRetryNoRatelimit(ctx context.Context, resp *http.Response, err error) (bool, error) {
	shouldRetry, checkErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if checkErr != nil {
		return false, checkErr
	}

	if shouldRetry && resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		shouldRetry = false
	}

	return shouldRetry, nil
}

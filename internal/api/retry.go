package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// requestFunc creates and executes an HTTP request.
type requestFunc func(ctx context.Context) (*http.Response, error)

// responseHandler processes a successful HTTP response.
type responseHandler[T any] func(resp *http.Response) (T, error)

// doWithRetry executes a request with retry logic.
func doWithRetry[T any](
	ctx context.Context,
	c *client,
	reqFn requestFunc,
	handleFn responseHandler[T],
) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; ; attempt++ {
		// Check context before making request
		if err := ctx.Err(); err != nil {
			return zero, fmt.Errorf("context error: %w", err)
		}

		resp, err := reqFn(ctx)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			if c.shouldRetry(err, 0, attempt) {
				if sleepErr := sleep(ctx, c.calculateBackoff(attempt)); sleepErr != nil {
					return zero, sleepErr
				}
				continue
			}
			return zero, lastErr
		}

		statusCode := resp.StatusCode

		if !isSuccessStatus(statusCode) {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if readErr != nil {
				lastErr = &APIError{
					StatusCode: statusCode,
					Message:    fmt.Sprintf("failed to read error body: %v", readErr),
				}
			} else {
				lastErr = &APIError{
					StatusCode: statusCode,
					Body:       string(body),
				}
			}

			if c.shouldRetry(nil, statusCode, attempt) {
				if sleepErr := sleep(ctx, c.calculateBackoff(attempt)); sleepErr != nil {
					return zero, sleepErr
				}
				continue
			}
			return zero, lastErr
		}

		return handleFn(resp)
	}
}


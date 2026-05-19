package middleware

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const (
	idempTTL          = 24 * time.Hour
	idempLockValue    = "__processing__"
	idempPollInterval = 50 * time.Millisecond
	idempPollTimeout  = 10 * time.Second
)

type cachedResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

type bodyDumpResponseWriter struct {
	*echo.Response
	body *bytes.Buffer
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.Response.Write(b)
}

func Idempotency(redisClient *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			idempKey := c.Request().Header.Get("X-Idempotency-Key")
			if idempKey == "" {
				return next(c)
			}

			redisKey := "idemp:" + idempKey
			ctx := c.Request().Context()

			// Atomic lock acquisition using SetNX.
			// If the key doesn't exist, we acquire the lock and proceed.
			// If the key already exists, another request is either processing or has completed.
			acquired, err := redisClient.SetNX(ctx, redisKey, idempLockValue, idempTTL).Result()
			if err != nil {
				// Redis error — fail open, let the request through
				log.Printf("[IDEMPOTENCY] Redis SetNX error: %v, proceeding without idempotency", err)
				return next(c)
			}

			if !acquired {
				// Key already exists — either processing or completed
				return waitForResult(c, redisClient, redisKey)
			}

			// We acquired the lock — process the request
			res := &bodyDumpResponseWriter{
				Response: c.Response(),
				body:     bytes.NewBuffer(nil),
			}
			c.Response().Writer = res

			handlerErr := next(c)

			// Cache 2xx and 4xx responses; on 5xx, delete the lock so retries can proceed
			if res.Status >= 200 && res.Status < 500 {
				headers := make(map[string]string)
				for k, v := range res.Header() {
					if len(v) > 0 {
						headers[k] = v[0]
					}
				}

				cr := cachedResponse{
					StatusCode: res.Status,
					Headers:    headers,
					Body:       res.body.Bytes(),
				}

				crBytes, marshalErr := json.Marshal(cr)
				if marshalErr == nil {
					redisClient.Set(ctx, redisKey, string(crBytes), idempTTL)
				} else {
					log.Printf("[IDEMPOTENCY] Failed to marshal response: %v", marshalErr)
					redisClient.Del(ctx, redisKey)
				}
			} else {
				// Server error — release the lock so the client can retry
				redisClient.Del(ctx, redisKey)
			}

			return handlerErr
		}
	}
}

// waitForResult polls Redis until the processing request completes and returns the cached response.
// If the poll times out, returns 409 Conflict.
func waitForResult(c echo.Context, redisClient *redis.Client, redisKey string) error {
	ctx := c.Request().Context()
	deadline := time.Now().Add(idempPollTimeout)

	for time.Now().Before(deadline) {
		val, err := redisClient.Get(ctx, redisKey).Result()
		if err != nil {
			// Key was deleted (server error on first request) — let caller retry
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "request was processing but failed, please retry with a new idempotency key",
			})
		}

		if val == idempLockValue {
			// Still processing, wait and poll again
			time.Sleep(idempPollInterval)
			continue
		}

		// We have a cached result
		var resp cachedResponse
		if json.Unmarshal([]byte(val), &resp) == nil {
			for k, v := range resp.Headers {
				c.Response().Header().Set(k, v)
			}
			c.Response().Header().Set("X-Cache", "HIT")
			return c.Blob(resp.StatusCode, c.Response().Header().Get(echo.HeaderContentType), resp.Body)
		}

		// Corrupted cache — delete and let retry
		redisClient.Del(ctx, redisKey)
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "cached response corrupted, please retry with a new idempotency key",
		})
	}

	// Timeout waiting for the other request to finish
	return c.JSON(http.StatusConflict, map[string]string{
		"error": "request is still being processed, please try again later",
	})
}

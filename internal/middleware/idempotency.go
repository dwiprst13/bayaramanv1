package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
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

			cached, err := redisClient.Get(ctx, redisKey).Result()
			if err == nil && cached != "" {
				var resp cachedResponse
				if json.Unmarshal([]byte(cached), &resp) == nil {
					for k, v := range resp.Headers {
						c.Response().Header().Set(k, v)
					}
					c.Response().Header().Set("X-Cache", "HIT")
					return c.Blob(resp.StatusCode, c.Response().Header().Get(echo.HeaderContentType), resp.Body)
				}
			}

			// Wrap response writer
			res := &bodyDumpResponseWriter{
				Response: c.Response(),
				body:     bytes.NewBuffer(nil),
			}
			c.Response().Writer = res

			err = next(c)

			// Cache 2xx and 4xx responses
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
					redisClient.Set(context.Background(), redisKey, string(crBytes), 24*time.Hour)
				} else {
					log.Printf("Failed to marshal idempotency response: %v", marshalErr)
				}
			}

			return err
		}
	}
}

package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// NewLogger creates a new middleware handler
func NewLogger() fiber.Handler {
	// Set variables
	var (
		start, stop time.Time
		once        sync.Once
		errHandler  fiber.ErrorHandler
	)

	// Return new handler
	return func(c *fiber.Ctx) (err error) {
		// Set error handler once
		once.Do(func() {
			// override error handler
			errHandler = c.App().Config().ErrorHandler
		})

		// Set latency start time
		start = time.Now()

		// Handle request, store err for logging
		chainErr := c.Next()

		// Manually call error handler
		if chainErr != nil {
			if err := errHandler(c, chainErr); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		// Set latency stop time
		stop = time.Now()

		entry := log.WithFields(log.Fields{
			"StatusCode":        c.Response().StatusCode(),
			"Latency":           stop.Sub(start).Round(time.Millisecond),
			"IP":                c.IP(),
			"Method":            c.Method(),
			"Path":              c.Path(),
			"Referer":           c.Get(fiber.HeaderReferer),
			"Protocol":          c.Protocol(),
			"XForwardedFor":     c.Get(fiber.HeaderXForwardedFor),
			"Host":              c.Hostname(),
			"URL":               c.OriginalURL(),
			"UserAgent":         c.Get(fiber.HeaderUserAgent),
			"NumBytesReceived":  len(c.Request().Body()),
			"NumBytesSent":      len(c.Response().Body()),
			"Route":             c.Route().Path,
			"RequestBody":       string(c.Body()),
			"QueryStringParams": c.Request().URI().QueryArgs().String(),
		})

		code := c.Response().StatusCode()
		switch {
		case code >= fiber.StatusOK && code < fiber.StatusMultipleChoices:
			entry.Info("Processed HTTP request")
		case code >= fiber.StatusMultipleChoices && code < fiber.StatusBadRequest:
			entry.Info("Forward HTTP request")
		case code >= fiber.StatusBadRequest && code < fiber.StatusInternalServerError:
			entry.Warn("Bad HTTP request")
		default:
			entry.Error("Internal Server Error")
		}

		return nil
	}
}

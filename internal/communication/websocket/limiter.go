package websocket

import (
	"strings"
	"strconv"
	"math"
	"bytes"
	"github.com/throttled/throttled"

	core "github.com/miphilipp/devchat-server/internal"
)

type WebsocketVaryBy struct {
    RemoteAddr bool
	Method bool
	Ressource bool
}

// Key returns the key for this request based on the criteria defined by the VaryBy struct.
func (vb *WebsocketVaryBy) Key(method int, ressource string, remoteAddr string) string {
	var buf bytes.Buffer
	sep := "\n" // Separator defaults to newline

	buf.WriteString("ws__")

	if vb.RemoteAddr && len(remoteAddr) > 0 {
		index := strings.LastIndex(remoteAddr, ":")

		var ip string
		if index == -1 {
			ip = remoteAddr
		} else {
			ip = remoteAddr[:index]
		}

		buf.WriteString(strings.ToLower(ip) + sep)
	}
	if vb.Method {
		buf.WriteString(strconv.Itoa(method) + sep)
	}
	if vb.Ressource {
		buf.WriteString(ressource + sep)
	}
	return buf.String()
}

type WebsocketRateLimiter struct {
	RateLimiter throttled.RateLimiter
	VaryBy interface {
		Key(method int, ressource string, remoteAddr string) string
	}
}

func newWebsocketRateLimiter(store throttled.GCRAStore, vary *WebsocketVaryBy, perMin, burstSize int) (*WebsocketRateLimiter, error) {
	quota := throttled.RateQuota{throttled.PerMin(perMin), burstSize}
	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return nil, err
	}

	return &WebsocketRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      vary,
	}, nil
}

// RateLimit checks if the rate limit for the given citeria was reached. If yes it returns
// an RequestLimitExceededError error as its first return value. The second returned error is not nil
// if something went wrong.
func (l *WebsocketRateLimiter) RateLimit(method int, ressource string, remoteAddr string) (error, error) {
	k := l.VaryBy.Key(method, ressource, remoteAddr)
	limited, context, err := l.RateLimiter.RateLimit(k, 1)
	if err != nil {
		return nil, err
	}

	if limited {
		var limit int
		if v := context.Limit; v >= 0 {
			limit = v
		}
	
		var remaining int
		if v := context.Remaining; v >= 0 {
			remaining = v
		}
	
		var resetAfter int
		if v := context.ResetAfter; v >= 0 {
			resetAfter = int(math.Ceil(v.Seconds()))
		}
	
		var retryAfter int
		if v := context.RetryAfter; v >= 0 {
			retryAfter = int(math.Ceil(v.Seconds()))
		}

		return core.NewRequestLimitExceededError(retryAfter, remaining, limit, resetAfter), nil
	}

	return nil, nil
}
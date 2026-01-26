package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimiter IP 频率限制器
type IPRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.Mutex
	rate     int           // 每秒允许请求数
	burst    int           // 允许突发请求数
}

// Visitor 访问者
type Visitor struct {
	timestamp time.Time
	remaining int
}

// NewIPRateLimiter 创建新的频率限制器
func NewIPRateLimiter(rate, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
	}
}

// Middleware 返回 Gin 中间件
func (r *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 清理过期记录
		r.mu.Lock()
		defer r.mu.Unlock()

		now := time.Now()
		for ip, visitor := range r.visitors {
			if now.Sub(visitor.timestamp) > time.Second*time.Duration(r.burst)/time.Duration(r.rate) {
				delete(r.visitors, ip)
			}
		}

		// 获取客户端IP
		ip := c.ClientIP()

		// 检查访问记录
		visitor, exists := r.visitors[ip]
		if !exists {
			r.visitors[ip] = &Visitor{
				timestamp: now,
				remaining: r.burst - 1,
			}
		} else {
			elapsed := now.Sub(visitor.timestamp)
			visitor.timestamp = now
			visitor.remaining = r.burst - 1

			// 根据时间恢复配额
			if elapsed > 0 {
				visitor.remaining += int(elapsed.Seconds()) * r.rate
				if visitor.remaining > r.burst {
					visitor.remaining = r.burst
				}
			}
		}

		// 检查剩余配额
		if visitor.remaining < 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"ok":      false,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", string(rune(r.burst)))
		c.Header("X-RateLimit-Remaining", string(rune(visitor.remaining)))

		c.Next()
	}
}

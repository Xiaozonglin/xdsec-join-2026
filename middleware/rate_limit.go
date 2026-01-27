package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimiter IP 频率限制器
type IPRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.Mutex
	rate     float64 // 每秒恢复的配额数
	burst    int     // 初始配额（最大配额）
}

// Visitor 访问者
type Visitor struct {
	timestamp time.Time
	remaining float64
}

// NewIPRateLimiter 创建新的频率限制器
// rate: 每秒恢复的配额数（例如 1 表示每秒恢复1个，0.05 表示每20秒恢复1个，即每分钟3个）
// burst: 初始配额和最大配额
func NewIPRateLimiter(rate float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
	}
}

// Middleware 返回 Gin 中间件
func (r *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if r.rate <= 0 || r.burst <= 0 {
			c.Next()
			return
		}

		// 清理过期记录
		r.mu.Lock()
		defer r.mu.Unlock()

		now := time.Now()
		for ip, visitor := range r.visitors {
			// 清理超过恢复周期的记录
			if now.Sub(visitor.timestamp) > time.Duration(float64(r.burst)/r.rate)*time.Second {
				delete(r.visitors, ip)
			}
		}

		// 获取客户端IP
		ip := c.ClientIP()

		// 检查访问记录
		visitor, exists := r.visitors[ip]
		if !exists {
			visitor = &Visitor{
				timestamp: now,
				remaining: float64(r.burst),
			}
			r.visitors[ip] = visitor
		} else {
			elapsed := now.Sub(visitor.timestamp)
			visitor.timestamp = now

			// 根据时间恢复配额
			if elapsed > 0 {
				recovered := elapsed.Seconds() * r.rate
				visitor.remaining += recovered
				if visitor.remaining > float64(r.burst) {
					visitor.remaining = float64(r.burst)
				}
			}
		}

		// 消耗一个配额
		visitor.remaining -= 1

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
		c.Header("X-RateLimit-Limit", strconv.Itoa(r.burst))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(int(math.Floor(visitor.remaining))))

		c.Next()
	}
}

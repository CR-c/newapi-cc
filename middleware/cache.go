package middleware

import (
	"github.com/gin-gonic/gin"
)

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		if requestPath == "/" ||
			requestPath == "/docs.html" ||
			requestPath == "/docs-official.html" ||
			requestPath == "/docs.css" ||
			requestPath == "/docs.js" {
			c.Header("Cache-Control", "no-cache, must-revalidate")
		} else {
			c.Header("Cache-Control", "max-age=604800") // one week
		}
		c.Header("Cache-Version", "b688f2fb5be447c25e5aa3bd063087a83db32a288bf6a4f35f2d8db310e40b14")
		c.Next()
	}
}

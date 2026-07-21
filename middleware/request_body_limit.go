package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var imageEditRequestSlots = make(chan struct{}, 2)

func AnonymousRequestBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		maxBytes := common.GetAnonymousRequestBodyLimitBytes()
		if maxBytes <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}

		originalBody := c.Request.Body
		limitedBody, err := readAnonymousRequestBody(originalBody, maxBytes)
		_ = originalBody.Close()
		if err != nil {
			if common.IsRequestBodyTooLargeError(err) {
				c.AbortWithStatus(http.StatusRequestEntityTooLarge)
				return
			}
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(limitedBody))
		c.Request.ContentLength = int64(len(limitedBody))
		c.Next()
	}
}

func ImageEditRequestBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		isImageEdit := path == "/v1/images/edits" || path == "/pg/images/edits"
		if !isImageEdit || !strings.Contains(c.Request.Header.Get("Content-Type"), gin.MIMEMultipartPOSTForm) {
			c.Next()
			return
		}
		if c.Request.ContentLength > constant.MaxImageEditMultipartBytes {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		select {
		case imageEditRequestSlots <- struct{}{}:
			defer func() { <-imageEditRequestSlots }()
		case <-c.Request.Context().Done():
			c.AbortWithStatus(499)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, constant.MaxImageEditMultipartBytes)
		c.Next()
	}
}

func readAnonymousRequestBody(body io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, common.ErrRequestBodyTooLarge
	}
	return data, nil
}

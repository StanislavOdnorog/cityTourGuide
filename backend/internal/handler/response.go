package handler

import "github.com/gin-gonic/gin"

func errorJSON(c *gin.Context, status int, message string) {
	c.JSON(status, errorBody(c, message))
}

func errorJSONWithFields(c *gin.Context, status int, message string, fields gin.H) {
	c.JSON(status, errorBodyWithFields(c, message, fields))
}

func errorBody(c *gin.Context, message string) gin.H {
	body := gin.H{"error": message}
	if traceID, ok := c.Get("trace_id"); ok {
		if value, ok := traceID.(string); ok && value != "" {
			body["request_id"] = value
		}
	}
	return body
}

func errorBodyWithFields(c *gin.Context, message string, fields gin.H) gin.H {
	body := errorBody(c, message)
	for k, v := range fields {
		body[k] = v
	}
	return body
}

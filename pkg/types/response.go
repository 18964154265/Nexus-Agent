package types

import (
    "github.com/cloudwego/hertz/pkg/app"
)

type Response struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data"`
    RequestID string      `json:"request_id"`
}

func Write(c *app.RequestContext, code int, message string, data interface{}) {
    rid := string(c.GetHeader("X-Request-ID"))
    c.JSON(200, &Response{Code: code, Message: message, Data: data, RequestID: rid})
}


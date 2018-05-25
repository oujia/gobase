package gobase

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Result int `json:"result"`
	Code int `json:"code"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

func NewResponse(baseError *BaseError, data interface{}) *Response {
	result := 1
	if baseError.Code() != 0 {
		result = 0
	}

	return &Response{Result:result, Code:baseError.Code(), Msg:baseError.Error(), Data:data}
}

func NewResponseWithMSG(baseError *BaseError, data interface{}, message string) *Response {
	result := 1
	if baseError.Code() > 0 {
		result = 0
	}
	msg := message
	if len(msg) == 0 {
		msg = baseError.Error()
	}

	return &Response{Result:result, Code:baseError.Code(), Msg:msg, Data:data}
}

func (resp *Response) SendBy(c *gin.Context) {
	c.JSON(http.StatusOK, resp)
}
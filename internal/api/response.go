package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK   = 1
	CodeFail = 0
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

type BusinessError struct {
	Msg  string
	Data interface{}
}

func (e *BusinessError) Error() string { return e.Msg }

func Fail(msg string) *BusinessError { return &BusinessError{Msg: msg} }

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Msg: "ok", Data: data})
}

func FailJSON(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{Code: CodeFail, Msg: msg})
}

func HandleBusiness(c *gin.Context, err error, data interface{}) {
	if err == nil {
		OK(c, data)
		return
	}
	if be, ok := err.(*BusinessError); ok {
		c.JSON(http.StatusOK, Response{Code: CodeFail, Msg: be.Msg, Data: be.Data})
		return
	}
	c.JSON(http.StatusOK, Response{Code: CodeFail, Msg: err.Error()})
}

func NotImplemented(c *gin.Context) {
	FailJSON(c, "NOT_IMPLEMENTED")
}

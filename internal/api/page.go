package api

import "github.com/gin-gonic/gin"

// PageResult is the standard list envelope payload.
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int         `json:"total"`
	PageNum  int         `json:"page_num,omitempty"`
	PageSize int         `json:"page_size,omitempty"`
}

func PageOK(c *gin.Context, list interface{}, total, pageNum, pageSize int) {
	OK(c, PageResult{List: list, Total: total, PageNum: pageNum, PageSize: pageSize})
}

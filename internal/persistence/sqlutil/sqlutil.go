package sqlutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// NowExpr returns SQL expression for current timestamp.
func NowExpr(driver string) string {
	if strings.EqualFold(driver, "mysql") {
		return "NOW()"
	}
	return "datetime('now')"
}

func Page(c *gin.Context) (pageNum, pageSize int) {
	pageNum, _ = strconv.Atoi(c.DefaultQuery("page_num", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return
}

func Offset(pageNum, pageSize int) int {
	return (pageNum - 1) * pageSize
}

func LimitSQL(driver string, limit, offset int) string {
	if strings.EqualFold(driver, "mysql") {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}

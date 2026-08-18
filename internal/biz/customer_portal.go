package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
	"erp/internal/middleware"
)

const ctxBoundCustomerID = "bound_customer_id"

func (s *Services) boundCustomerID(userID int64) (int64, bool) {
	if s == nil || s.DB == nil || userID <= 0 {
		return 0, false
	}
	var cid int64
	err := s.DB.QueryRow(`SELECT COALESCE(customer_id,0) FROM iam_user WHERE id=? AND COALESCE(is_deleted,0)=0`, userID).Scan(&cid)
	return cid, err == nil && cid > 0
}

func portalCustomerID(c *gin.Context) (int64, bool) {
	if c == nil {
		return 0, false
	}
	v, ok := c.Get(ctxBoundCustomerID)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, t > 0
	case int:
		return int64(t), t > 0
	default:
		return 0, false
	}
}

func isCustomerClient(c *gin.Context) bool {
	cl := middleware.Claims(c)
	return cl != nil && cl.ClientType == "customer"
}

func (s *Services) bindPortalCustomer(c *gin.Context) bool {
	if !isCustomerClient(c) {
		return true
	}
	cl := middleware.Claims(c)
	cid, ok := s.boundCustomerID(cl.UserID)
	if !ok {
		api.FailJSON(c, "CUSTOMER_NOT_BOUND")
		return false
	}
	c.Set(ctxBoundCustomerID, cid)
	return true
}

func (s *Services) refusePortalMismatch(c *gin.Context, customerID int64) bool {
	cid, ok := portalCustomerID(c)
	if !ok {
		return false
	}
	if customerID <= 0 || customerID != cid {
		api.FailJSON(c, "NOT_FOUND")
		return true
	}
	return false
}

func applyPortalCustomerSQL(c *gin.Context, col string, where *string, args *[]interface{}) {
	cid, ok := portalCustomerID(c)
	if !ok || col == "" {
		return
	}
	*where += ` AND ` + col + `=?`
	*args = append(*args, cid)
}

func (s *Services) orderCustomerID(orderID int64) int64 {
	if orderID <= 0 {
		return 0
	}
	var cid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(customer_id,0) FROM sl_sales_order WHERE id=? AND COALESCE(is_deleted,0)=0`, orderID).Scan(&cid)
	return cid
}

func (s *Services) deliveryCustomerID(deliveryID int64) int64 {
	var oid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(order_id,0) FROM sl_delivery_approval WHERE id=?`, deliveryID).Scan(&oid)
	return s.orderCustomerID(oid)
}

func (s *Services) preShipCustomerID(preShipID int64) int64 {
	var oid int64
	_ = s.DB.QueryRow(`SELECT COALESCE(order_id,0) FROM sl_pre_shipment WHERE id=?`, preShipID).Scan(&oid)
	return s.orderCustomerID(oid)
}

func ginHInt64(v interface{}) int64 {
	n, _ := asInt64(v)
	return n
}

func customerSalesAllowed(method, openapiPath, action string) bool {
	p := openapiPath
	if p == "" {
		return false
	}
	switch {
	case strings.HasPrefix(p, "/api/v1/sales/inquiries"):
		if strings.Contains(p, "/approve") || strings.Contains(p, "/reject") || strings.Contains(p, "/to-order") ||
			action == "action:approve" || action == "action:reject" || action == "action:to-order" {
			return false
		}
		return action == "list" || action == "get" || action == "create" || action == "update" || action == "replace" ||
			strings.Contains(p, "/submit") || strings.Contains(p, "/withdraw") ||
			action == "action:submit" || action == "action:withdraw"
	case strings.HasPrefix(p, "/api/v1/sales/self-orders"):
		return method == "GET" || method == "POST" || action == "list" || action == "create"
	case strings.HasPrefix(p, "/api/v1/sales/my-orders"):
		return action == "list" || action == "get"
	case strings.HasPrefix(p, "/api/v1/sales/orders"):
		return action == "list" || action == "get" || strings.Contains(p, "/rebuy") || action == "action:rebuy"
	case strings.HasPrefix(p, "/api/v1/sales/deliveries"):
		return action == "list" || action == "get"
	case strings.HasPrefix(p, "/api/v1/sales/pre-shipments"):
		return action == "list" || action == "get"
	case strings.HasPrefix(p, "/api/v1/sales/price-locks"):
		return action == "list" || action == "get"
	case strings.HasPrefix(p, "/api/v1/sales/contracts"):
		return action == "list" || action == "get"
	case strings.HasPrefix(p, "/api/v1/sales/quote-histories"):
		return true
	case strings.HasPrefix(p, "/api/v1/sales/quote-calculator"):
		return true
	default:
		return false
	}
}

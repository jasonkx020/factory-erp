package biz

import "testing"

func TestCustomerSalesAllowed(t *testing.T) {
	allow := []struct{ method, path, action string }{
		{"GET", "/api/v1/sales/inquiries", "list"},
		{"POST", "/api/v1/sales/inquiries", "create"},
		{"POST", "/api/v1/sales/inquiries/{id}/submit", "action:submit"},
		{"POST", "/api/v1/sales/inquiries/{id}/withdraw", "action:withdraw"},
		{"POST", "/api/v1/sales/self-orders", "create"},
		{"GET", "/api/v1/sales/my-orders", "list"},
		{"POST", "/api/v1/sales/orders/{id}/rebuy", "action:rebuy"},
		{"GET", "/api/v1/sales/deliveries", "list"},
		{"GET", "/api/v1/sales/quote-histories", "list"},
		{"POST", "/api/v1/sales/quote-calculator/apply", "action:apply"},
	}
	for _, c := range allow {
		if !customerSalesAllowed(c.method, c.path, c.action) {
			t.Fatalf("want allow %s %s %s", c.method, c.path, c.action)
		}
	}
	deny := []struct{ method, path, action string }{
		{"POST", "/api/v1/sales/inquiries/{id}/approve", "action:approve"},
		{"POST", "/api/v1/sales/inquiries/{id}/to-order", "action:to-order"},
		{"POST", "/api/v1/sales/deliveries/{id}/approve", "action:approve"},
		{"POST", "/api/v1/sales/orders", "create"},
		{"GET", "/api/v1/sales/rankings", "list"},
		{"POST", "/api/v1/sales/self-order-rules", "create"},
	}
	for _, c := range deny {
		if customerSalesAllowed(c.method, c.path, c.action) {
			t.Fatalf("want deny %s %s %s", c.method, c.path, c.action)
		}
	}
}

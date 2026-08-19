package biz

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Services) productMeta(id int64) (name, category, code string) {
	if id <= 0 {
		return "", "", ""
	}
	_ = s.DB.QueryRow(`SELECT COALESCE(name,''), COALESCE(category,''), COALESCE(code,'') FROM prd_product WHERE id=?`, id).
		Scan(&name, &category, &code)
	return name, category, code
}

func (s *Services) enrichBoxCodeRows(list []map[string]interface{}) {
	cache := map[int64]gin.H{}
	for _, row := range list {
		pid := asInt64Or0(row["product_id"])
		meta, ok := cache[pid]
		if !ok {
			name, cat, code := s.productMeta(pid)
			meta = gin.H{"product_name": name, "product_category": cat, "product_code": code}
			cache[pid] = meta
		}
		row["product_name"] = meta["product_name"]
		row["product_category"] = meta["product_category"]
		row["product_code"] = meta["product_code"]
	}
}

func (s *Services) productStockFromBoxes() []gin.H {
	rows, err := s.DB.Query(`SELECT COALESCE(b.product_id,0), COALESCE(MAX(p.name),''), COALESCE(MAX(p.category),''), COALESCE(MAX(p.code),''),
		COUNT(1),
		COALESCE(SUM(CASE WHEN LOWER(COALESCE(b.status,'')) NOT IN ('destroyed','void','finished') THEN COALESCE(b.weight, b.qty, 0) ELSE 0 END),0)
		FROM inv_box_code b
		LEFT JOIN prd_product p ON p.id=b.product_id
		WHERE COALESCE(b.is_deleted,0)=0
		GROUP BY COALESCE(b.product_id,0)
		ORDER BY COALESCE(MAX(p.name), '')`)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var pid, boxCnt int64
		var name, cat, code string
		var remain float64
		if err := rows.Scan(&pid, &name, &cat, &code, &boxCnt, &remain); err != nil {
			continue
		}
		out = append(out, gin.H{
			"product_id": pid, "product_name": name, "product_category": cat, "product_code": code,
			"box_count": boxCnt, "remain_kg": remain, "qty": remain,
		})
	}
	return out
}

func (s *Services) summarizeTraceBoxProgress() []gin.H {
	lotNet := map[string]float64{}
	lotTickets := map[string]int{}
	lotStatus := map[string]string{}
	q, err := s.DB.Query(`SELECT UPPER(COALESCE(trace_code,'')), COALESCE(SUM(net_weight),0), COUNT(1),
		MAX(CASE WHEN LOWER(COALESCE(status,''))='gate_accepted' THEN 'gate_accepted' ELSE COALESCE(status,'') END)
		FROM pur_weigh_ticket
		WHERE COALESCE(is_deleted,0)=0 AND COALESCE(trace_code,'')<>''
		AND LOWER(COALESCE(status,'')) IN ('gate_accepted','stocked')
		GROUP BY UPPER(COALESCE(trace_code,''))`)
	if err == nil {
		defer q.Close()
		for q.Next() {
			var trace, st string
			var net float64
			var n int
			if err := q.Scan(&trace, &net, &n, &st); err != nil {
				continue
			}
			lotNet[trace] = net
			lotTickets[trace] = n
			lotStatus[trace] = st
		}
	}
	rows, err := s.DB.Query(`SELECT UPPER(COALESCE(b.trace_code,'')), COALESCE(b.product_id,0),
		COALESCE(MAX(p.name),''), COALESCE(MAX(p.category),''),
		COUNT(1), COALESCE(SUM(COALESCE(b.weight, b.qty, 0)),0)
		FROM inv_box_code b
		LEFT JOIN prd_product p ON p.id=b.product_id
		WHERE COALESCE(b.is_deleted,0)=0 AND COALESCE(b.trace_code,'')<>''
		AND LOWER(COALESCE(b.status,'')) NOT IN ('destroyed','void')
		GROUP BY UPPER(COALESCE(b.trace_code,'')), COALESCE(b.product_id,0)
		ORDER BY 1`)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()
	seen := map[string]bool{}
	out := []gin.H{}
	for rows.Next() {
		var trace, name, cat string
		var pid int64
		var qty int
		var boxed float64
		if err := rows.Scan(&trace, &pid, &name, &cat, &qty, &boxed); err != nil {
			continue
		}
		seen[trace] = true
		net := lotNet[trace]
		remain := net - boxed
		if remain < 0 {
			remain = 0
		}
		complete := net > 0 && (remain <= 5 || boxed+0.0001 >= net)
		if strings.ToLower(lotStatus[trace]) == "stocked" {
			complete = true
		}
		out = append(out, gin.H{
			"trace_code": trace, "product_id": pid, "product_name": name, "product_category": cat,
			"boxed_qty": qty, "boxed_weight": boxed, "lot_net_weight": net,
			"ticket_count": lotTickets[trace], "remaining_weight": remain, "complete": complete,
		})
	}
	for trace, net := range lotNet {
		if seen[trace] {
			continue
		}
		out = append(out, gin.H{
			"trace_code": trace, "product_id": 0, "product_name": "", "product_category": "",
			"boxed_qty": 0, "boxed_weight": 0.0, "lot_net_weight": net,
			"ticket_count": lotTickets[trace], "remaining_weight": net, "complete": false,
		})
	}
	return out
}

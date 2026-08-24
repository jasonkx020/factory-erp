package biz

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type routingStepProduct struct {
	StepID          int64
	SeqNo           int
	ProcessID       int64
	OutputProductID int64
}

// resolveRoutingIDForTrace prefers locked trace session routing, else product default.
func (s *Services) resolveRoutingIDForTrace(trace string, productID int64) int64 {
	trace = strings.ToUpper(strings.TrimSpace(trace))
	if rid := s.resolveTraceSessionRoutingID(trace); rid > 0 {
		return rid
	}
	return s.resolveRoutingID(0, productID)
}

func (s *Services) loadRoutingStepProducts(routingID int64) []routingStepProduct {
	if routingID <= 0 {
		return nil
	}
	rows, err := s.DB.Query(`SELECT id, seq_no, process_id, COALESCE(output_product_id,0)
		FROM pd_routing_step WHERE routing_id=? ORDER BY seq_no, id`, routingID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []routingStepProduct{}
	for rows.Next() {
		var st routingStepProduct
		if err := rows.Scan(&st.StepID, &st.SeqNo, &st.ProcessID, &st.OutputProductID); err != nil {
			continue
		}
		out = append(out, st)
	}
	return out
}

func (s *Services) routingHasOutputProducts(routingID int64) bool {
	for _, st := range s.loadRoutingStepProducts(routingID) {
		if st.OutputProductID > 0 {
			return true
		}
	}
	return false
}

func (s *Services) resolveInputProductForStep(routingID int64, step *routingStepProduct) int64 {
	if step == nil || routingID <= 0 {
		return 0
	}
	steps := s.loadRoutingStepProducts(routingID)
	for i, st := range steps {
		if st.StepID == step.StepID || (st.ProcessID == step.ProcessID && st.SeqNo == step.SeqNo) {
			if i == 0 {
				return 0
			}
			return steps[i-1].OutputProductID
		}
	}
	return 0
}

func (s *Services) resolveStepByOutputProduct(routingID, productID int64) *routingStepProduct {
	if routingID <= 0 || productID <= 0 {
		return nil
	}
	for _, st := range s.loadRoutingStepProducts(routingID) {
		if st.OutputProductID == productID {
			cp := st
			return &cp
		}
	}
	return nil
}

func (s *Services) resolveNextProcessByProduct(routingID, productID int64) int64 {
	st := s.resolveStepByOutputProduct(routingID, productID)
	if st == nil {
		return 0
	}
	steps := s.loadRoutingStepProducts(routingID)
	for i, cur := range steps {
		if cur.StepID == st.StepID {
			if i+1 < len(steps) {
				return steps[i+1].ProcessID
			}
			return 0
		}
	}
	return 0
}

func (s *Services) resolveProcessProducts(routingID, processID int64) (inputProductID, outputProductID int64) {
	if routingID <= 0 || processID <= 0 {
		return 0, 0
	}
	steps := s.loadRoutingStepProducts(routingID)
	for i, st := range steps {
		if st.ProcessID != processID {
			continue
		}
		outputProductID = st.OutputProductID
		if i > 0 {
			inputProductID = steps[i-1].OutputProductID
		}
		return inputProductID, outputProductID
	}
	return 0, 0
}

func (s *Services) resolveStepOutputProduct(routingID, processID int64) int64 {
	_, out := s.resolveProcessProducts(routingID, processID)
	return out
}

func (s *Services) resolveInboundStepByProduct(routingID, productID int64) *routingStep {
	if routingID <= 0 || productID <= 0 {
		return nil
	}
	var stepID int64
	_ = s.DB.QueryRow(`SELECT id FROM pd_routing_step
		WHERE routing_id=? AND output_product_id=? AND COALESCE(is_inbound_checkpoint,0)=1
		ORDER BY seq_no LIMIT 1`, routingID, productID).Scan(&stepID)
	if stepID <= 0 {
		_ = s.DB.QueryRow(`SELECT id FROM pd_routing_step
			WHERE routing_id=? AND output_product_id=?
			ORDER BY seq_no LIMIT 1`, routingID, productID).Scan(&stepID)
	}
	if stepID <= 0 {
		return nil
	}
	return s.loadStep(stepID)
}

func (s *Services) validateRoutingOutputProducts(routingID int64, steps []compiledStep, finalProductID int64) string {
	if len(steps) == 0 {
		return ""
	}
	seen := map[int64]bool{}
	hasAny := false
	for _, st := range steps {
		if st.OutputProductID <= 0 {
			continue
		}
		hasAny = true
		if seen[st.OutputProductID] {
			return "ROUTING_OUTPUT_PRODUCT_DUPLICATE"
		}
		seen[st.OutputProductID] = true
	}
	if !hasAny {
		return ""
	}
	last := steps[len(steps)-1]
	if last.OutputProductID <= 0 {
		return "ROUTING_FINAL_OUTPUT_REQUIRED"
	}
	if finalProductID > 0 && last.OutputProductID != finalProductID {
		return "ROUTING_FINAL_PRODUCT_MISMATCH"
	}
	for i, st := range steps {
		if st.OutputProductID <= 0 {
			return fmt.Sprintf("ROUTING_STEP_OUTPUT_REQUIRED:%s", st.Code)
		}
		if i > 0 && steps[i-1].OutputProductID > 0 && st.OutputProductID == steps[i-1].OutputProductID {
			return "ROUTING_OUTPUT_PRODUCT_DUPLICATE"
		}
	}
	return ""
}

func (s *Services) assertBoardProductForProcess(board *boardState, processID int64) string {
	if board == nil || processID <= 0 {
		return ""
	}
	routingID := s.resolveRoutingIDForTrace(board.Trace, board.ProductID)
	if routingID <= 0 || !s.routingHasOutputProducts(routingID) {
		return ""
	}
	inputPID, _ := s.resolveProcessProducts(routingID, processID)
	expected := inputPID
	if expected <= 0 {
		expected = s.resolveTraceProductID(board.Trace)
	}
	if expected <= 0 {
		return ""
	}
	if board.ProductID != expected {
		return "PRODUCT_PROCESS_MISMATCH"
	}
	return ""
}

func (s *Services) routingMatchesTraceProduct(routingID, traceProductID int64) bool {
	if routingID <= 0 || traceProductID <= 0 {
		return false
	}
	var rProductID int64
	_ = s.DB.QueryRow(`SELECT COALESCE(product_id,0) FROM pd_routing WHERE id=? AND COALESCE(is_deleted,0)=0`, routingID).Scan(&rProductID)
	if rProductID == traceProductID {
		return true
	}
	var firstOut int64
	_ = s.DB.QueryRow(`SELECT COALESCE(output_product_id,0) FROM pd_routing_step WHERE routing_id=? ORDER BY seq_no, id LIMIT 1`, routingID).Scan(&firstOut)
	return firstOut == traceProductID
}

func (s *Services) assertStockInProductForProcess(trace string, productID, processID int64) string {
	if processID <= 0 {
		return ""
	}
	routingID := s.resolveRoutingIDForTrace(trace, productID)
	if routingID <= 0 || !s.routingHasOutputProducts(routingID) {
		return ""
	}
	_, expected := s.resolveProcessProducts(routingID, processID)
	if expected <= 0 {
		return ""
	}
	if productID != expected {
		return "PRODUCT_STEP_MISMATCH"
	}
	return ""
}

func (s *Services) attachProcessProductPreview(dst gin.H, routingID, processID int64) {
	if dst == nil || routingID <= 0 || processID <= 0 {
		return
	}
	inputPID, outputPID := s.resolveProcessProducts(routingID, processID)
	if inputPID > 0 {
		_, inName, _ := s.productMeta(inputPID)
		dst["input_product_id"] = inputPID
		dst["input_product_name"] = inName
	}
	if outputPID > 0 {
		_, outName, _ := s.productMeta(outputPID)
		dst["output_product_id"] = outputPID
		dst["output_product_name"] = outName
	}
}

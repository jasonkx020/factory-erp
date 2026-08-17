package biz

import "testing"

func TestCarrierCodeLabels(t *testing.T) {
	l, s, m, split := CarrierCodeLabels("board")
	if l != "板码" || s != "板" || m != "板码管理" || split != "分板" {
		t.Fatalf("board labels: %s %s %s %s", l, s, m, split)
	}
	l, s, m, split = CarrierCodeLabels("box")
	if l != "箱码" || s != "箱" || m != "箱码管理" || split != "分箱" {
		t.Fatalf("box labels: %s %s %s %s", l, s, m, split)
	}
	if normalizeCarrierCodeUnitValue("") != "board" {
		t.Fatal("empty should default board")
	}
	if normalizeCarrierCodeUnitValue("BOX") != "box" {
		t.Fatal("BOX should normalize to box")
	}
	out := map[string]interface{}{"carrier_code_unit": "box"}
	enrichCarrierCodeLabels(out)
	if out["carrier_code_label"] != "箱码" || out["carrier_code_manage_title"] != "箱码管理" {
		t.Fatalf("enrich: %+v", out)
	}
}

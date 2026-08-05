package biz

import "github.com/gin-gonic/gin"

// settleAmount = net*price + freight + loading + weigh
func settleAmount(net, unitPrice, freight, loading, weigh float64) (goods, total float64) {
	goods = net * unitPrice
	total = goods + freight + loading + weigh
	return goods, total
}

func feeFieldsFromBody(body map[string]interface{}) (freight, loading, weigh, passRate, reject float64, plate, recvAddr string) {
	freight, _ = asFloat(body["freight_fee"])
	loading, _ = asFloat(body["loading_fee"])
	weigh, _ = asFloat(body["weigh_fee"])
	passRate, _ = asFloat(body["pass_rate"])
	reject, _ = asFloat(body["reject_weight"])
	plate = strOr(body["plate_no"])
	recvAddr = strOr(body["receive_address"])
	return
}

func feeMap(freight, loading, weigh, passRate, reject float64, plate, recvAddr string) gin.H {
	return gin.H{
		"freight_fee": freight, "loading_fee": loading, "weigh_fee": weigh,
		"pass_rate": passRate, "reject_weight": reject,
		"plate_no": plate, "receive_address": recvAddr,
	}
}

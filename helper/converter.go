package helper

import (
	"encoding/json"
	"strconv"
)

func ToInt(v interface{}) int {
	switch vv := v.(type) {
	case float64:
		return int(vv)
	case string:
		n, _ := strconv.Atoi(vv)
		return n
	case json.Number:
		n, _ := vv.Int64()
		return int(n)
	case int:
		return vv
	case int64:
		return int(vv)
	default:
		return 0
	}
}

func ToFloat64(v interface{}) float64 {
	switch vv := v.(type) {
	case float64:
		return vv
	case string:
		n, _ := strconv.ParseFloat(vv, 64)
		return n
	case json.Number:
		n, _ := vv.Float64()
		return n
	case int:
		return float64(vv)
	case int64:
		return float64(vv)
	default:
		return 0
	}
}

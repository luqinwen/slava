package sortedset

import (
	"errors"
	"strconv"
)

// 这段代码实现了一个 ScoreBorder 结构体，用于表示 ZRANGEBYSCORE 命令中的 min 和 max 参数，包括 <, <=, >, >=, +inf 和 -inf。
// 同时，提供了 ParseScoreBorder 函数，用于将字符串解析为 ScoreBorder 对象。
//
// ScoreBorder 包含三个字段，分别为 Inf、Value 和 Exclude，表示范围的上下限、具体的值以及是否包含上下限。
// greater 函数用于判断某个数值是否在 ScoreBorder 的上限范围之内，less 函数则用于判断是否在下限范围之内。
// 代码中还定义了 positiveInfBorder 和 negativeInfBorder 两个变量，用于表示正无穷和负无穷。
// ParseScoreBorder 函数接收一个字符串作为参数，根据字符串的内容返回一个 ScoreBorder 对象。
// 如果字符串表示正负无穷，则返回 positiveInfBorder 或 negativeInfBorder；
// 如果字符串以 ( 开头，则表示排除这个值，即 ScoreBorder.Exclude 为 true；
// 否则表示包含这个值，ScoreBorder.Exclude 为 false。
const (
	negativeInf int8 = -1
	positiveInf int8 = 1
)

// ScoreBorder represents range of a float value, including: <, <=, >, >=, +inf, -inf
type ScoreBorder struct {
	Inf     int8
	Value   float64
	Exclude bool
}

// if max.greater(score) then the score is within the upper border
// do not use min.greater()
func (border *ScoreBorder) greater(value float64) bool {
	if border.Inf == negativeInf {
		return false
	} else if border.Inf == positiveInf {
		return true
	}
	if border.Exclude {
		return border.Value > value
	}
	return border.Value >= value
}

// border<value
func (border *ScoreBorder) less(value float64) bool {
	if border.Inf == negativeInf {
		return true
	} else if border.Inf == positiveInf {
		return false
	}
	if border.Exclude {
		return border.Value < value
	}
	return border.Value <= value
}

var positiveInfBorder = &ScoreBorder{
	Inf: positiveInf,
}

var negativeInfBorder = &ScoreBorder{
	Inf: negativeInf,
}

// ParseScoreBorder creates ScoreBorder from slava arguments
func ParseScoreBorder(s string) (*ScoreBorder, error) {
	if s == "inf" || s == "+inf" {
		return positiveInfBorder, nil
	}
	if s == "-inf" {
		return negativeInfBorder, nil
	}
	if s[0] == '(' {
		value, err := strconv.ParseFloat(s[1:], 64)
		if err != nil {
			return nil, errors.New("ERR min or max is not a float")
		}
		return &ScoreBorder{
			Inf:     0,
			Value:   value,
			Exclude: true,
		}, nil
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, errors.New("ERR min or max is not a float")
	}
	return &ScoreBorder{
		Inf:     0,
		Value:   value,
		Exclude: false,
	}, nil
}

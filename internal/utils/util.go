package utils

import (
	"errors"
	"strconv"
)

func MergeMaps[K comparable, V any](m1, m2 map[K]V) map[K]V {
	merged := make(map[K]V, len(m1)+len(m2))

	for k, v := range m1 {
		merged[k] = v
	}

	for k, v := range m2 {
		merged[k] = v
	}

	return merged
}

func SplitStatus(target string) ([]int, error) {
	l := len(target)

	if l%3 != 0 {
		return nil, errors.New("请求参数不合法,每三位为一个状态码")
	} else {
		status := make([]int, 0, l/3)
		for i := 0; i <= l-3; i += 3 {
			if s, err := strconv.Atoi(target[i : i+3]); err != nil {
				return nil, errors.New("请求参数不合法,请输入有效状态码")
			} else {
				status = append(status, s)
			}
		}
		return status, nil
	}

}

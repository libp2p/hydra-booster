package utils

import (
	"errors"
	"strings"
)

func ParseOptsString(s string) (map[string]string, error) {
	m := map[string]string{}
	opts := strings.Split(s, ",")
	for _, opt := range opts {
		entry := strings.Split(opt, "=")
		if len(entry) != 2 {
			return nil, errors.New("option config must be key=value pairs")
		}
		m[entry[0]] = entry[1]
	}
	return m, nil
}

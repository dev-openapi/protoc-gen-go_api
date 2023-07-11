package genapi

import (
	"errors"
	"fmt"
	"strings"
)

type options struct {
	// 输出文件路径
	out string
}

func parseOptions(param *string) (*options, error) {
	opts := options{}
	if param == nil {
		return nil, errors.New("empty options parameter")
	}
	for _, s := range strings.Split(*param, ",") {
		if s == "" {
			continue
		}
		e := strings.IndexByte(s, '=')
		if e < 0 {
			return nil, fmt.Errorf("invalid plugin option format, must be key=value: %s", s)
		}
		key, val := s[:e], s[e+1:]
		if val == "" {
			return nil, fmt.Errorf("invalid plugin option value, missing value in key=value: %s", s)
		}

		switch key {
		case "out":
			opts.out = val
		}
	}
	return &opts, nil
}

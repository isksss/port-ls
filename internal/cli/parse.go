package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var stateInputRE = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)

func parsePortArg(v string) (int, error) {
	p, err := strconv.Atoi(v)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("port must be 1-65535: %s", v)
	}
	return p, nil
}

func parseRangeArg(v string) (int, int, error) {
	if strings.Contains(v, "-") {
		parts := strings.SplitN(v, "-", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return 0, 0, fmt.Errorf("range must be start-end")
		}
		start, err := parsePortArg(parts[0])
		if err != nil {
			return 0, 0, err
		}
		end, err := parsePortArg(parts[1])
		if err != nil {
			return 0, 0, err
		}
		if end < start {
			return 0, 0, fmt.Errorf("range end must be greater than or equal to start")
		}
		return start, end, nil
	}
	start, err := parsePortArg(v)
	if err != nil {
		return 0, 0, err
	}
	end := start + 1000
	if end > 65535 {
		end = 65535
	}
	return start, end, nil
}

func normalizeStateInput(v string) (string, error) {
	n := strings.Trim(strings.ToLower(v), " ")
	n = strings.NewReplacer("-", "_", " ", "_").Replace(n)
	n = strings.Trim(n, "_")
	if n == "" || !stateInputRE.MatchString(n) {
		return "", fmt.Errorf("invalid state: %s", v)
	}
	return n, nil
}

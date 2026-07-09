package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.(0|[1-9][0-9]*))?$`)

type Version struct {
	Major int
	Minor int
	Patch int
	RC    int
}

func Validate(raw string) error {
	_, err := Parse(raw)
	return err
}

func Parse(raw string) (Version, error) {
	match := versionPattern.FindStringSubmatch(raw)
	if match == nil {
		return Version{}, fmt.Errorf("invalid version %q: expected vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-rc.N", raw)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	rc := 0
	if match[5] != "" {
		parsedRC, _ := strconv.Atoi(match[5])
		rc = parsedRC
	}
	return Version{Major: major, Minor: minor, Patch: patch, RC: rc}, nil
}

func (v Version) String() string {
	base := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.RC > 0 {
		return fmt.Sprintf("%s-rc.%d", base, v.RC)
	}
	return base
}

func NextStable(raw string) (string, error) {
	if raw == "" {
		return "v0.1.0", nil
	}
	parsed, err := Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.RC > 0 {
		parsed.RC = 0
		return parsed.String(), nil
	}
	parsed.Patch++
	return parsed.String(), nil
}

func NextRC(raw string) (string, error) {
	if raw == "" {
		return "v0.1.0-rc.1", nil
	}
	parsed, err := Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.RC > 0 {
		parsed.RC++
		return parsed.String(), nil
	}
	parsed.Patch++
	parsed.RC = 1
	return parsed.String(), nil
}

func Compare(leftRaw, rightRaw string) int {
	left, leftErr := Parse(leftRaw)
	right, rightErr := Parse(rightRaw)
	if leftErr != nil && rightErr != nil {
		return 0
	}
	if leftErr != nil {
		return -1
	}
	if rightErr != nil {
		return 1
	}
	return compareParsed(left, right)
}

func TrimTagPrefix(tag string) string {
	if tag == "" {
		return ""
	}
	idx := strings.LastIndex(tag, "/")
	if idx == -1 {
		return tag
	}
	return tag[idx+1:]
}

func compareParsed(left, right Version) int {
	if left.Major != right.Major {
		return cmpInt(left.Major, right.Major)
	}
	if left.Minor != right.Minor {
		return cmpInt(left.Minor, right.Minor)
	}
	if left.Patch != right.Patch {
		return cmpInt(left.Patch, right.Patch)
	}
	if left.RC == right.RC {
		return 0
	}
	if left.RC == 0 {
		return 1
	}
	if right.RC == 0 {
		return -1
	}
	return cmpInt(left.RC, right.RC)
}

func cmpInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

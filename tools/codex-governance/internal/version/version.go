// Package version parses and compares Semantic Versioning 2.0.0 values.
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var expression = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

type Version struct {
	Major, Minor, Patch uint64
	Pre, Build          string
}

func Parse(value string) (Version, error) {
	m := expression.FindStringSubmatch(value)
	if m == nil {
		return Version{}, fmt.Errorf("invalid Semantic Versioning 2.0.0 value %q", value)
	}
	for _, part := range strings.Split(m[4], ".") {
		if part != "" && numericLeadingZero(part) {
			return Version{}, fmt.Errorf("invalid prerelease identifier %q", part)
		}
	}
	major, _ := strconv.ParseUint(m[1], 10, 64)
	minor, _ := strconv.ParseUint(m[2], 10, 64)
	patch, _ := strconv.ParseUint(m[3], 10, 64)
	return Version{Major: major, Minor: minor, Patch: patch, Pre: m[4], Build: m[5]}, nil
}
func numericLeadingZero(value string) bool {
	_, err := strconv.ParseUint(value, 10, 64)
	return err == nil && len(value) > 1 && value[0] == '0'
}
func (v Version) String() string {
	value := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		value += "-" + v.Pre
	}
	if v.Build != "" {
		value += "+" + v.Build
	}
	return value
}

// Compare returns SemVer precedence. Build metadata is intentionally ignored.
func (v Version) Compare(other Version) int {
	for _, pair := range [][2]uint64{{v.Major, other.Major}, {v.Minor, other.Minor}, {v.Patch, other.Patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if v.Pre == "" && other.Pre != "" {
		return 1
	}
	if v.Pre != "" && other.Pre == "" {
		return -1
	}
	if v.Pre == other.Pre {
		return 0
	}
	left, right := strings.Split(v.Pre, "."), strings.Split(other.Pre, ".")
	for i := 0; i < len(left) && i < len(right); i++ {
		a, b := left[i], right[i]
		an, ae := strconv.ParseUint(a, 10, 64)
		bn, be := strconv.ParseUint(b, 10, 64)
		if ae == nil && be == nil {
			if an < bn {
				return -1
			}
			if an > bn {
				return 1
			}
			continue
		}
		if ae == nil {
			return -1
		}
		if be == nil {
			return 1
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	return 1
}
func (v Version) Next(impact string) (Version, error) {
	switch impact {
	case "none":
		return v, nil
	case "major":
		if v.Major == ^uint64(0) {
			return Version{}, fmt.Errorf("major version overflow")
		}
		return Version{Major: v.Major + 1}, nil
	case "minor":
		if v.Minor == ^uint64(0) {
			return Version{}, fmt.Errorf("minor version overflow")
		}
		return Version{Major: v.Major, Minor: v.Minor + 1}, nil
	case "patch":
		if v.Patch == ^uint64(0) {
			return Version{}, fmt.Errorf("patch version overflow")
		}
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}, nil
	default:
		return Version{}, fmt.Errorf("impact must be major, minor, patch, or none")
	}
}

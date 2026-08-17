package dbmigrate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var upgradeFilePattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)_(.+)\.sql$`)

// Version is a semantic migration version like v1.0.1.
type Version struct {
	Raw    string
	Major  int
	Minor  int
	Patch  int
	Slug   string
	Source string
}

func ParseVersion(raw string) (Version, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "v") {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	parts := strings.Split(strings.TrimPrefix(raw, "v"), ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q", raw)
	}
	return Version{Raw: raw, Major: major, Minor: minor, Patch: patch}, nil
}

func ParseUpgradeFilename(name string) (Version, error) {
	m := upgradeFilePattern.FindStringSubmatch(filepath.Base(name))
	if m == nil {
		return Version{}, fmt.Errorf("invalid upgrade filename %q", name)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	raw := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	return Version{
		Raw:    raw,
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Slug:   m[4],
		Source: name,
	}, nil
}

func CompareVersions(a, b Version) int {
	if a.Major != b.Major {
		return cmpInt(a.Major, b.Major)
	}
	if a.Minor != b.Minor {
		return cmpInt(a.Minor, b.Minor)
	}
	return cmpInt(a.Patch, b.Patch)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func SortVersions(items []Version) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if CompareVersions(items[j], items[i]) < 0 {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

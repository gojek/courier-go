package courier

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionString(t *testing.T) {
	semver := regexp.MustCompile(`^(?P<major>\d+\.)?(?P<minor>\d+\.)?(?P<patch>\*|\d+)(?:-(?P<mod>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?$`)
	assert.True(t, semver.MatchString(Version()))
}

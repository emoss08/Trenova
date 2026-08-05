package runpattern

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromSelectsExactlyTheNamedTests(t *testing.T) {
	pattern := From([]string{"TestB", "TestA"})

	assert.Equal(t, "^(TestA|TestB)$", pattern)

	re := regexp.MustCompile(pattern)
	assert.True(t, re.MatchString("TestA"))
	assert.False(t, re.MatchString("TestAB"), "an anchored pattern must not select by prefix")
	assert.False(t, re.MatchString("XTestA"))
}

func TestFromQuotesMetacharacters(t *testing.T) {
	pattern := From([]string{"Test$Weird.Name"})

	re, err := regexp.Compile(pattern)
	require.NoError(t, err)
	assert.True(t, re.MatchString("Test$Weird.Name"))
	assert.False(t, re.MatchString("Test$WeirdXName"), "the dot must be literal, not any-character")
}

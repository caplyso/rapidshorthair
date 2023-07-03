package rapidshorthair

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSizeAndCheckRangeSupport(t *testing.T) {
	_, err := GetSizeAndCheckRangeSupport("http://www.baidu.com")
	require.NotEqual(t, err, nil, "")

	_, err = GetSizeAndCheckRangeSupport("https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz")
	require.Equal(t, err, nil, "")
}

func TestGetFileName(t *testing.T) {
	s, err := GetFileName("https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz")
	require.Equal(t, err, nil, "")
	require.Equal(t, s, "go1.20.5.darwin-amd64.tar.gz", "")
}

func TestGetRangeBody(t *testing.T) {
	_, l, err := GetRangeBody("https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz", 0, 102400)
	require.Equal(t, l, int64(102401), "")
	require.Equal(t, err, nil, "")
}

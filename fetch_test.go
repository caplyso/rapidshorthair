package rapidshorthair

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetcher(t *testing.T) {
	f, err := NewFetcher("https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz",
		3, 2, 0,
		"/home/go1.20.5.darwin-amd64.tar.gz",
		false)
	require.Equal(t, err, nil, "")
	require.NotEqual(t, f, nil, "")

	err = f.Start()
	require.Equal(t, err, nil, "")
}

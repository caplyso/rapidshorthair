package rapidshorthair

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	response, err := http.Get("http://www.baidu.com/")
	require.Equal(t, err, nil, "")
	require.NotEqual(t, response.Body, nil, "")

	buf := make([]byte, 4096)

	s := &segment{
		buf: buf,
	}

	l, err := s.read(response.Body)
	require.NotEqual(t, l, 0, "")
	require.Equal(t, err, nil, "")
}

func TestWrite(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	require.Equal(t, err, nil, "")

	buf := make([]byte, 12)
	s := &segment{
		buf:    buf,
		length: 12,
	}

	l, err := s.writeTo(devNull)
	require.NotEqual(t, l, 0, "")
	require.Equal(t, err, nil, "")
}

package rapidshorthair

import (
	"fmt"
	"io"
	"os"
)

type segment struct {
	buf    []byte
	length int64
	offset int64
}

func (s *segment) writeTo(f *os.File) (int64, error) {
	buf := s.buf[:s.length]
	nw, err := f.WriteAt(buf, s.offset)
	if err != nil {
		return -1, err
	}

	if s.length != int64(nw) {
		return -1, fmt.Errorf("recv length mismatch %v %v", s.length, nw)
	}

	return int64(nw), nil
}

func (s *segment) read(reader io.ReadCloser) (int64, error) {
	nr, err := reader.Read(s.buf)
	if err != nil {
		return -1, err
	}

	s.length = int64(nr)

	return int64(nr), nil
}

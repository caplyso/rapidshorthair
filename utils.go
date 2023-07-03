package rapidshorthair

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
)

func GetSizeAndCheckRangeSupport(url string) (size int64, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	res, err := client.Do(req)
	if err != nil {
		return
	}

	header := res.Header
	accept_ranges, supported := header["Accept-Ranges"]
	if !supported {
		return 0, errors.New("Doesn't support header `Accept-Ranges`.")
	} else if supported && accept_ranges[0] != "bytes" {
		return 0, errors.New("Support `Accept-Ranges`, but value is not `bytes`.")
	}

	size, err = strconv.ParseInt(header["Content-Length"][0], 10, 64)

	return
}

func GetFileName(download_url string) (string, error) {
	url_struct, err := url.Parse(download_url)
	if err != nil {
		return "", err
	}

	return filepath.Base(url_struct.Path), nil
}

func GetRangeBody(url string, start int64, end int64) (io.ReadCloser, int64, error) {
	var client http.Client
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	return resp.Body, size, err
}

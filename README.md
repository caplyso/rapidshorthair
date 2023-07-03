# rapidshorthair

[![Go Report Card](https://goreportcard.com/badge/github.com/caplyso/rapidshorthair)](https://goreportcard.com/report/github.com/caplyso/rapidshorthair)
[![Go Reference](https://pkg.go.dev/badge/github.com/caplyso/rapidshorthair.svg)](https://pkg.go.dev/github.com/caplyso/rapidshorthair)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

A high-performance concurrent downloader.

# Overview
In order to speed up the download speed of large files, a very simple idea is to open multiple parallel download instances. In the process of downloading, you also need to consider the allocation of buffers, the process of writing files, and so on. In order to facilitate the process of speeding up file downloads in various projects, the process of downloading files in parallel is implemented as a common library.

This library implements the following features:
- A generic work queue.
- Parallel Download Framework.
- A simple memory pool.
- File Parallel Write Framework.

# Example
A simple use case is as follow.
```
package main

import (
	"github.com/caplyso/rapidshorthair"
)

func main() {
	f, err := NewFetcher("https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz",
		3, 2, 0,
		"/home/go1.20.5.darwin-amd64.tar.gz",
		false)
    if err != nil {
        return
    }

	f.Start()
}

```

## Contributing
- Fork it
- Create your feature branch (`git checkout -b my-new-feature`)
- Commit your changes (`git commit -am 'Add some feature'`)
- Push to the branch (`git push origin my-new-feature`)
- Create new Pull Request

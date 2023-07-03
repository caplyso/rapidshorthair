package rapidshorthair

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type Fetcher struct {
	url string

	concurrency int
	batches     int
	bufferSz    int64

	File *os.File

	filePath     string
	withDateTime bool

	wm *WorkerManager

	totalSize   int64
	recvedTotal int64
}

func NewFetcher(url string, concurrency int, batches int, bufferSz int64,
	filePath string, withDateTime bool) (*Fetcher, error) {
	if concurrency == 0 {
		concurrency = 2
	}

	if batches == 0 {
		batches = 2
	}

	if bufferSz == 0 {
		bufferSz = 4 * 1024 * 1024
	}

	file_size, err := GetSizeAndCheckRangeSupport(url)
	if err != nil {
		return nil, err
	}

	return &Fetcher{
		url:          url,
		totalSize:    file_size,
		concurrency:  concurrency,
		batches:      batches,
		bufferSz:     bufferSz,
		filePath:     filePath,
		withDateTime: withDateTime,
	}, nil
}

func (f *Fetcher) TotalLength() int64 {
	return atomic.LoadInt64(&f.totalSize)
}

func (f *Fetcher) CurrentLength() int64 {
	return atomic.LoadInt64(&f.recvedTotal)
}

func (f *Fetcher) Start() error {
	var file_path string
	var err error
	if len(f.filePath) > 0 {
		file_path = f.filePath
	} else {
		file_path, err = GetFileName(f.url)
		if err != nil {
			return err
		}
	}

	if f.withDateTime {
		file_path = file_path + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	fd, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	wm, err := NewWorkerManager("segment", 0, nil)
	if err != nil {
		return err
	}

	var start, end int64
	var partial_size = int64(f.totalSize / int64(f.concurrency))
	for i := 0; i < f.concurrency; i++ {
		if i == f.concurrency-1 {
			end = f.totalSize
		} else {
			end = start + partial_size
		}

		w, err := NewSegmentCrawler(int(i), fd, f.url, f.batches, f.bufferSz, start, end-1)
		if err != nil {
			return err
		}

		wm.Add(w)

		start = end
	}

	f.wm = wm

	return f.Run()
}

func (f *Fetcher) Run() error {
	wctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(1 * time.Second)
	stopChan := make(chan bool)

	go func(ticker *time.Ticker) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count := f.wm.Count()
				var recvTotal int64 = 0
				for i := int32(0); i < count; i++ {
					w, err := f.wm.GetWorker(i)
					if err != nil {
						return
					}

					sz, err := w.Downcall(nil)
					if err != nil {
						return
					}

					recvTotal += sz.(int64)
				}

				atomic.StoreInt64(&f.recvedTotal, recvTotal)
				if recvTotal >= f.totalSize {
					cancel()
					return
				}
			case stop := <-stopChan:
				if stop {
					cancel()
					return
				}
			}
		}
	}(ticker)

	ch := make(chan error, 1)
	go func(ch chan error) {
		f.wm.Run(wctx, ch)
	}(ch)

	err := <-ch
	return err
}

type bufferPair struct {
	recvBufferChan    chan *segment
	releaseBufferChan chan *segment
}

func NewBufferPair(channels int, sz int64, init bool) *bufferPair {
	recvChan := make(chan *segment, channels+1)
	releaseChan := make(chan *segment, channels+1)

	if init {
		for i := 0; i < channels; i++ {
			buf := make([]byte, sz)

			s := &segment{
				buf: buf,
			}

			releaseChan <- s
		}
	}

	return &bufferPair{
		recvBufferChan:    recvChan,
		releaseBufferChan: releaseChan,
	}
}

type SegmentCrawler struct {
	name        string
	index       int
	bufferIndex int32
	bufferList  []*bufferPair
	RangeBatch  *WorkerManager
	url         string
	start       int64
	end         int64

	totalSize   int64
	recvedTotal int64
}

func NewSegmentCrawler(index int, f *os.File, url string,
	batches int, bufferSz int64,
	start int64, end int64) (*SegmentCrawler, error) {
	sc := &SegmentCrawler{
		name:       fmt.Sprintf("segment %d", index),
		index:      index,
		start:      start,
		end:        end,
		url:        url,
		bufferList: make([]*bufferPair, 0),
	}

	rb, err := NewWorkerManager("batch", batches, func() (Worker, error) {
		bp := NewBufferPair(batches, bufferSz, true)
		sc.bufferList = append(sc.bufferList, bp)

		return NewBatchCrawler(f, bp.recvBufferChan, bp.releaseBufferChan, func(sz int64) {
			atomic.AddInt64(&sc.recvedTotal, sz)
		})
	})

	if err != nil {
		return nil, err
	}

	sc.RangeBatch = rb

	return sc, nil
}

func (c *SegmentCrawler) Identity() string {
	return c.name
}

func (c *SegmentCrawler) Status() bool {
	return true
}

func (c *SegmentCrawler) Downcall(_ interface{}) (interface{}, error) {
	return atomic.LoadInt64(&c.recvedTotal), nil
}

func (c *SegmentCrawler) getRecvBuffer() *segment {
	for {
		if c.bufferIndex >= c.RangeBatch.Count() {
			c.bufferIndex = 0
		}

		w, err := c.RangeBatch.GetWorker(c.bufferIndex)
		if err != nil {
			return nil
		}

		if !w.Status() {
			s := <-c.bufferList[c.bufferIndex].releaseBufferChan
			c.bufferIndex += 1
			return s
		}

		c.bufferIndex += 1
	}
}

func (c *SegmentCrawler) dispatch(reader io.ReadCloser, offset int64) (int64, error) {
	s := c.getRecvBuffer()
	nr, err := s.read(reader)

	if err != nil {
		return -1, err
	}

	s.offset = offset

	for {
		if c.bufferIndex >= c.RangeBatch.Count() {
			c.bufferIndex = 0
		}

		w, err := c.RangeBatch.GetWorker(c.bufferIndex)
		if err != nil {
			return int64(nr), fmt.Errorf("worker fetch failed, index = %v", c.bufferIndex)
		}

		if !w.Status() {
			c.bufferList[c.bufferIndex].recvBufferChan <- s
			return int64(nr), nil
		}

		c.bufferIndex += 1
	}
}

func (c *SegmentCrawler) wait(cancel context.CancelFunc) {
	recvedTotal := atomic.LoadInt64(&c.recvedTotal)
	if recvedTotal >= c.totalSize {
		cancel()
		return
	}

	ticker := time.NewTimer(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		recvedTotal := atomic.LoadInt64(&c.recvedTotal)
		if recvedTotal >= c.totalSize {
			cancel()
			return
		}
	}
}

func (c *SegmentCrawler) finish() error {
	return nil
}

func (c *SegmentCrawler) Run(ctx context.Context) error {
	body, size, err := GetRangeBody(c.url, c.start, c.end)
	if err != nil {
		return err
	}

	c.totalSize = size

	bctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go c.RangeBatch.Run(bctx, ch)

	offset := c.start

	for {
		select {
		case <-ctx.Done():
			cancel()
			return nil
		case err := <-ch:
			cancel()
			return err
		default:
			nr, err := c.dispatch(body, offset)
			if nr > 0 {
				offset += int64(nr)
			}

			if err != nil {
				if err.Error() == "EOF" {
					c.wait(cancel)
				}

				return c.finish()
			}
		}
	}
}

type BatchCrawler struct {
	name string
	File *os.File

	recvBufferChan    chan *segment
	releaseBufferChan chan *segment

	update func(int64)

	status int64
}

func NewBatchCrawler(f *os.File,
	recvBufferChan chan *segment,
	releaseBufferChan chan *segment, update func(int64)) (*BatchCrawler, error) {

	return &BatchCrawler{
		name:              "BatchCrawler",
		File:              f,
		recvBufferChan:    recvBufferChan,
		releaseBufferChan: releaseBufferChan,
		update:            update,
	}, nil
}

func (c *BatchCrawler) Identity() string {
	return c.name
}

func (c *BatchCrawler) bufferRelease(s *segment) {
	c.releaseBufferChan <- s
}

func (c *BatchCrawler) Status() bool {
	status := atomic.LoadInt64(&c.status)
	return status != 0
}

func (c *BatchCrawler) Downcall(_ interface{}) (interface{}, error) {
	return nil, nil
}

func (c *BatchCrawler) Run(ctx context.Context) error {
	for {
		select {
		case s := <-c.recvBufferChan:
			atomic.CompareAndSwapInt64(&c.status, 0, 1)
			nw, err := s.writeTo(c.File)
			if err != nil {
				return err
			}

			c.update(int64(nw))

			c.bufferRelease(s)
			atomic.CompareAndSwapInt64(&c.status, 1, 0)

		case <-ctx.Done():
			return nil
		}
	}
}

package util

import (
	"context"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"time"
)

// TimeoutPing wraps c.Ping() with a timeout.
func TimeoutPing(ctx context.Context, timeout time.Duration, c *elasticsearch.Client) (*esapi.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultCh := make(chan struct {
		res *esapi.Response
		err error
	}, 1)

	go func() {
		res, err := c.Ping()
		resultCh <- struct {
			res *esapi.Response
			err error
		}{res, err}
	}()

	select {
	case res := <-resultCh:
		return res.res, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

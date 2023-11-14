package main

import (
	"context"
	"flag"
	"os"
)

type conf struct {
	web struct {
		baseURL  string
		apiV2URL string
	}
}

func run(ctx context.Context, c *conf) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	apiV2URL := c.web.apiV2URL
	return doSomething(apiV2URL)
}

func main() {
	ctx := context.Background()
	c := conf{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fs.StringVar(&c.web.baseURL, "web-base-url", "", "base URL for website")
	fs.StringVar(&c.web.apiV2URL, "api-v2-url", "", "base APIv2 URL for website")

	if err := run(ctx, &c); err != nil {
		panic(err)
	}
}

func doSomething(_ string) error {
	return nil
}

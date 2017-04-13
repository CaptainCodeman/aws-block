# AWS Block

Middleware to block traffic coming from AWS networks by checking the various
[AWS address ranges](http://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html).

Sorry AWS, but all the annoying bots usually seem to come from you ...

If you have a web site then you will likely have it visited by bots, oh so many bots.
Some of them are useful and necessary and abide by `robots.txt` but many are badly bahaved
content scrapers, abusing your bandwidth, overloading your servers and costing you money
all so they can selling your content or use it to sell their own SEO services.

Amazon EC2 seems to be the go-to solution for anyone wanting to host a web crawler.
This package makes it easy to block this server-to-server traffic in order to prioritize
traffic from legitimate site visitors.

## Usage

Get package and import it:

    go get -u github.com/captaincodeman/aws-block

```go
import (
    "net/http"
    "golang.org/x/net/context"
    "github.com/captaincodeman/aws-block"
)
```

Create a new blocker instance configured as required:

```go
blocker := awsblock.New(&awsblock.Config{
    Interval: time.Hour * 24,   // periodic check for IP range updates
    Region:   "us-east-1",      // region to block (empty for all)
    Service:  "ec2",            // service to block (empty for all)
})
```

Start the updating service, pass in a cancellable-context if cancellation is required
(or else just use `context.Background()`):

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

blocker.Start(ctx, http.DefaultClient)
```

Use the blocker middleware to block any traffic coming from AWS:

```go
m := http.NewServeMux()
m.HandleFunc("/", index)
h := blocker.Middleware(m)

log.Fatal(http.ListenAndServe(":8080", h))
```

Or use the `AWSRequest` method within your own middleware:

```go
func LogAWS(blocker *awsblock.Blocker, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if blocker.AWSRequest(r) {
			log.Println("AWS request by", r.UserAgent())
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
```

It's a good idea to start by logging the requests that match so that you can
see which traffic _would_ be blocked if the middleware was used. If you have
some valid / legitimate traffic coming from AWS you may want to create some
whitelisting system first (e.g. by useragent, hostname or IP) to allow it.

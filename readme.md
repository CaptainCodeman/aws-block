# AWS Block

Middleware to block traffic coming from AWS networks by checking the various
[AWS address ranges](http://docs.aws.amazon.com/general/latest/gr/aws-ip-ranges.html)

Sorry AWS, but all the annoying bots usually seem to come from you ...

## Usage

Create a configuration:

```go
config := &awsblock.Config{
    Interval: time.Hour * 24,   // periodic check for updates
    Region:   "us-east-1",      // region to block
    Service:  "ec2",            // aws service to block
    Confirm:  func(w http.ResponseWriter, r *http.Request) bool {
        // whitelist IP / user-agent, log request etc...
        return true
    }
}
```

Create a new blocker instance using an http client and the config:

```go
blocker := New(config)
```

Start the updating service, pass in a cancellable-context if cancellation is required
or else just use `context.Background()`:

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

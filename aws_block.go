package awsblock

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
	"github.com/tomasen/realip"
)

const (
	url string = "https://ip-ranges.amazonaws.com/ip-ranges.json"
)

var (
	ErrNotChanged = errors.New("Not changed")
)

type (
	ipRanges struct {
		SyncToken  string   `json:"syncToken"`
		CreateDate string   `json:"createDate"`
		Prefixes   []prefix `json:"prefixes"`
	}

	prefix struct {
		IPPrefix string `json:"ip_prefix"`
		Region   string `json:"region"`
		Service  string `json:"service"`
	}

	Blocker struct {
		sync.RWMutex
		config *Config
		ipNets []*net.IPNet
	}

	Confirm func(w http.ResponseWriter, r *http.Request) bool

	Config struct {
		Interval time.Duration
		Region   string
		Service  string
		Confirm  Confirm
	}
)

func New(config *Config) *Blocker {
	if config.Confirm == nil {
		config.Confirm = func(w http.ResponseWriter, r *http.Request) bool {
			return true
		}
	}
	return &Blocker{
		config: config,
		ipNets: make([]*net.IPNet, 0),
	}
}

func (b *Blocker) Start(ctx context.Context, client *http.Client) {
	ticker := time.NewTicker(b.config.Interval)

	var (
		ipranges *ipRanges
		etag     string
		err      error
	)

	go func() {
		defer fmt.Println("stopped")

		for {
			if ipranges, etag, err = b.Request(client, etag); err == nil {
				b.Update(ipranges)
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (b *Blocker) Request(client *http.Client, etag string) (*ipRanges, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}

	etag = res.Header.Get("ETag")

	if res.StatusCode == http.StatusNotModified {
		return nil, etag, ErrNotChanged
	}

	defer res.Body.Close()

	var data ipRanges

	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&data); err != nil {
		return nil, "", err
	}

	return &data, etag, nil
}

func (b *Blocker) Update(ipranges *ipRanges) {
	b.Lock()
	defer b.Unlock()

	b.ipNets = make([]*net.IPNet, 0)

	for _, prefix := range ipranges.Prefixes {
		_, ipnet, err := net.ParseCIDR(prefix.IPPrefix)
		if err != nil {
			continue
		}

		if b.config.matches(prefix.Region, prefix.Service) {
			b.ipNets = append(b.ipNets, ipnet)
		}
	}
}

func (c *Config) matches(region, service string) bool {
	if strings.EqualFold(c.Region, region) && strings.EqualFold(c.Service, service) {
		return true
	}
	if c.Region == "" && strings.EqualFold(c.Service, service) {
		return true
	}
	if strings.EqualFold(c.Region, region) && c.Service == "" {
		return true
	}
	return false
}

func (b *Blocker) Middleware(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		userIP := realip.RealIP(r)
		ip := net.ParseIP(userIP)

		var block bool

		b.RLock()
		for _, net := range b.ipNets {
			if block = net.Contains(ip); block {
				break
			}
		}
		b.RUnlock()

		if block {
			// provide a chance for caller to
			// a) whitelist requests (e.g. allow certain IPs / user-agents)
			// b) log that some request is being blocked / set the response
			if block = b.config.Confirm(w, r); block {
				return
			}
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

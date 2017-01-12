package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/lox/httpcache"
	"github.com/lox/httpcache/httplog"
)

const (
	defaultOrigin = "http://127.0.0.1:80"
	defaultListen = "0.0.0.0:8080"
	defaultDir    = "./cachedata"
)

var (
	origin   string
	host     string
	listen   string
	useDisk  bool
	private  bool
	dir      string
	dumpHttp bool
	verbose  bool
)

func init() {
	flag.StringVar(&origin, "origin", defaultOrigin, "the origin url to proxy to")
	flag.StringVar(&host, "host", "", "the host header to send")
	flag.StringVar(&listen, "listen", defaultListen, "the host and port to bind to")
	flag.StringVar(&dir, "dir", defaultDir, "the dir to store cache data in, implies -disk")
	flag.BoolVar(&useDisk, "disk", false, "whether to store cache data to disk")
	flag.BoolVar(&verbose, "v", false, "show verbose output and debugging")
	flag.BoolVar(&private, "private", false, "make the cache private")
	flag.BoolVar(&dumpHttp, "dumphttp", false, "dumps http requests and responses to stdout")
	flag.Parse()

	if verbose {
		httpcache.DebugLogging = true
	}
}

func main() {
	u, err := url.Parse(origin)
	if err != nil {
		log.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Scheme = u.Scheme
			r.URL.Host = u.Host
			if strings.HasSuffix(r.URL.Path, "/") {
				r.URL.Path = path.Join(u.Path, r.URL.Path) + "/"
			} else {
				r.URL.Path = path.Join(u.Path, r.URL.Path)
			}
			if host != "" {
				r.Host = host
			}
		},
	}

	var cache httpcache.Cache

	if useDisk && dir != "" {
		log.Printf("storing cached resources in %s", dir)
		if err := os.MkdirAll(dir, 0700); err != nil {
			log.Fatal(err)
		}
		var err error
		cache, err = httpcache.NewDiskCache(dir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cache = httpcache.NewMemoryCache()
	}

	handler := httpcache.NewHandler(cache, proxy)
	handler.Shared = !private

	respLogger := httplog.NewResponseLogger(handler)
	respLogger.DumpRequests = dumpHttp
	respLogger.DumpResponses = dumpHttp
	respLogger.DumpErrors = dumpHttp

	log.Printf("listening on http://%s", listen)
	log.Fatal(http.ListenAndServe(listen, respLogger))
}

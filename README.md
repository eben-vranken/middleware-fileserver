# Middleware File Server

A static file server with a stack of middleware wrapped around it: structured request logging, basic auth, per-IP rate limiting, and CORS (with preflight handling). The file server itself is `http.FileServer` — three lines. The point is the middleware.

```
request → [logging] → [CORS] → [rateLimit] → [auth] → fileServer → response
```

## How to use

```sh
go run .
```

Server listens on `127.0.0.1:8080`. The `public/` directory holds the served files.

```sh
# Without credentials → 401 with WWW-Authenticate
$ curl -i http://localhost:8080/index.html

# With credentials → the file
$ curl -u user:admin http://localhost:8080/index.html

# Hit the rate limit
$ for i in $(seq 1 150); do curl -s -o /dev/null -u user:admin http://localhost:8080/; done

# Trigger a CORS preflight directly
$ curl -i -X OPTIONS http://localhost:8080/index.html \
    -H "Origin: http://example.com" \
    -H "Access-Control-Request-Method: GET" \
    -H "Access-Control-Request-Headers: authorization"
```

## What I learned

- **Middleware is just function composition.** A middleware takes an `http.Handler` and returns one that does work around a call to `next.ServeHTTP`. Stacking is nested function calls — no framework, no DI container.
- **Anything that satisfies `http.Handler` is interchangeable.** `http.FileServer`, `http.HandlerFunc`, a router, an embedded type — same shape, same wrappers.
- **`http.ResponseWriter` is write-only.** No `StatusCode()`, no `Body()`. To observe the status, wrap the writer in a struct that overrides `WriteHeader`. Every Go logging middleware does some version of this.
- **Pointer vs value receivers are in the method set.** A method on `*T` is only in the method set of `*T`. Pass the wrapper as a pointer or the override silently falls back to the embedded method.
- **Snapshot under the lock.** Copy shared data into a local *before* unlocking, then use the local downstream. Releasing the lock and re-acquiring it leaves a TOCTOU window.
- **`defer mu.Unlock()` matters for early returns.** Without `defer`, every error branch has to remember to unlock — and one of them will eventually forget.
- **`time.Ticker` + a goroutine is the canonical "do something every X".** `for range ticker.C { ... }` consumes the channel forever. Wrapping the work in another goroutine is unnecessary unless it's slow.
- **Browsers trigger basic auth dialogs on `401` + `WWW-Authenticate`.** Either alone does nothing. The combination is the spec-defined signal.
- **CORS preflight is the part that catches you out.** Same-origin testing never exercises the middleware. An `Authorization` header triggers a preflight `OPTIONS`, which has no creds — so CORS must sit *outside* auth in the chain.
- **Middleware order is a design decision.** Logging outermost (sees everything). CORS outside auth (preflight passes through). Rate limit before auth (so brute-force attempts are throttled).
- **Constant-time compare for credentials.** Plain `!=` on strings short-circuits at the first mismatched byte — a timing leak. Real auth uses `subtle.ConstantTimeCompare`.
- **Fixed-window rate limiting is the simplest thing that works.** Has a known edge-of-window weakness (`2×LIMIT` requests in two seconds), but it's two functions and a ticker. Sliding window and token bucket exist for reasons but aren't worth it here.

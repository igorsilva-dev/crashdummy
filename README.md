# crashdummy

A chaos-enabled mock backend: WireMock-style stubbing plus configurable latency and error injection, exposed over HTTP with Prometheus metrics.

crashdummy serves two kinds of routes from JSON config: **mappings** (canned responses for a method and path) and **proxies** (forwarders to an upstream URL). Any route can inject latency and errors, either from static config or toggled at runtime, so you can reproduce slow and failing dependencies on demand. It ships a `/metrics` endpoint so the faults it injects are observable.

## Quick start

Run locally from source:

```
go run .
# crashdummy listening at :10000
```

```
curl -s localhost:10000/health
# {"message": "Healthy !"}
```

Run the container:

```
docker build -t crashdummy .
docker run --rm -p 10000:10000 crashdummy
```

## Configuration

Config is loaded once at startup from three directories. crashdummy fails fast if any file is malformed, so a bad config never serves silently.

```
mappings/   one JSON file per stubbed route
proxies/    one JSON file per forwarded route
stubs/      response bodies referenced by mappings
```

The directory root defaults to the working directory and can be relocated with `CRASHDUMMY_CONFIG_DIR` (see [Environment variables](#environment-variables)).

### Mappings

A mapping serves a canned response for one method and path. Requests to the path with a different method get a `405`.

| Field                   | Type   | Required | Description                                              |
| ----------------------- | ------ | -------- | -------------------------------------------------------- |
| `request.method`        | string | yes      | HTTP method. Case-insensitive; defaults to `GET`.        |
| `request.url`           | string | yes      | Path to serve, for example `/orders`.                    |
| `response.status`       | int    | no       | Status code to return. Defaults to `200`.                |
| `response.bodyFileName` | string | yes      | File under `stubs/` whose contents are the response body. |
| `latencyInMilliseconds` | int    | no       | Base delay before responding. Defaults to `0`.           |
| `jitterInMilliseconds`  | int    | no       | Random delay added in `[-jitter, +jitter)`.              |
| `errorRate`             | float  | no       | Probability `[0,1]` of injecting an error instead.       |
| `errorStatus`           | int    | no       | Status to return when an error is injected. Defaults to `500`. |

`mappings/create-order.json`:

```json
{
  "request": { "method": "POST", "url": "/orders" },
  "response": { "status": 201, "bodyFileName": "created.json" }
}
```

### Stubs

A stub is the raw JSON body a mapping returns. It is validated as JSON at load time.

`stubs/created.json`:

```json
{ "status": "created", "id": "ord_demo_0001" }
```

### Proxies

A proxy forwards a path to an upstream URL, passing the upstream status and body back verbatim. If the upstream call fails, crashdummy returns `502` with a JSON error. Latency and error injection apply before the upstream is called, so an injected error short-circuits without touching the upstream.

| Field                   | Type   | Required | Description                                          |
| ----------------------- | ------ | -------- | ---------------------------------------------------- |
| `path`                  | string | yes      | Local path to forward.                               |
| `upstream`              | string | yes      | Upstream URL to call.                                |
| `method`                | string | yes      | Method used for the upstream request.                |
| `latencyInMilliseconds` | int    | no       | Base delay before forwarding. Defaults to `0`.       |
| `jitterInMilliseconds`  | int    | no       | Random delay added in `[-jitter, +jitter)`.          |
| `errorRate`             | float  | no       | Probability `[0,1]` of injecting an error instead.   |
| `errorStatus`           | int    | no       | Status to return when an error is injected. Defaults to `500`. |

`proxies/test-health.json`:

```json
{
  "path": "/test-health",
  "upstream": "http://localhost:10000/health",
  "method": "GET",
  "latencyInMilliseconds": 2000,
  "jitterInMilliseconds": 2000
}
```

## Fault injection

Every route carries a fault configuration: a base latency with jitter, and an error rate with a status. Set it statically in the route's JSON (the fields above), or change it at runtime.

### Runtime toggle

`POST /admin/chaos` retunes one route's faults without a restart, which is how you switch faults on during a demo. Address a route by its key: the `"METHOD /path"` pattern for a mapping, or the path for a proxy.

```
curl -s -X POST localhost:10000/admin/chaos \
  -d '{"route":"POST /orders","errorRate":1.0,"errorStatus":503}'
# {"chaos":{"latencyInMilliseconds":0,"jitterInMilliseconds":0,"errorRate":1,"errorStatus":503},"route":"POST /orders"}
```

```
curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost:10000/orders
# 503
```

Reset it by sending `errorRate` `0` (and `latencyInMilliseconds` `0`). The endpoint returns `400` on a bad body or missing `route`, and `404` for an unknown route. It is meant to stay internal, not exposed through any gateway.

### Response headers

Every response carries a `Chaos-Type` header describing how it was produced:

| Value   | Meaning                               |
| ------- | ------------------------------------- |
| `mock`  | A mapping's canned response.          |
| `proxy` | A proxied upstream response.          |
| `error` | An injected fault (from `errorRate`). |

## Metrics

`GET /metrics` serves Prometheus metrics, including the default Go and process collectors. Two crashdummy series are exposed:

| Metric                                | Type      | Labels                      |
| ------------------------------------- | --------- | --------------------------- |
| `crashdummy_requests_total`           | counter   | `route`, `method`, `status` |
| `crashdummy_request_duration_seconds` | histogram | `route`, `method`           |

The `status` label makes an injected error rate queryable directly, for example the fraction of `POST /orders` responses returning `5xx`.

## Environment variables

| Variable                | Default | Description                                              |
| ----------------------- | ------- | ------------------------------------------------------- |
| `PORT`                  | `10000` | Port to listen on.                                      |
| `CRASHDUMMY_CONFIG_DIR` | `.`     | Directory holding `mappings/`, `proxies/`, and `stubs/`. |

## Development

```
go test ./... -race    # unit tests
go vet ./...           # vet
gofmt -l .             # formatting (no output means clean)
```

CI runs golangci-lint, gosec, the race-enabled tests, a Docker build, and a trivy image scan on every pull request. Pushing a `v*` tag additionally publishes the image to `ghcr.io/igorsilva-dev/crashdummy`.

## Architecture

crashdummy is a standard `net/http` `ServeMux`. `Register` loads the config directories and registers one handler per route; each handler owns a `chaos.Chaos` value that applies latency and errors and can be retuned at runtime through the admin endpoint. A metrics middleware wraps the mux to record per-route counts and latencies. Configuration is read once at startup and validated eagerly, so misconfiguration surfaces immediately rather than at request time.

# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

A `v0.2.0` with a visibly worse default fault profile is planned to serve as the second version in the Phase 2 Istio canary demo.

## [0.1.0] - 2026-07-15

First release. A 2023 mock-server side project (`mock-be`), recovered and modernized into a chaos-enabled mock backend: WireMock-style stubbing plus configurable latency and error injection, deployed as the demo workload for the Kubernetes platform.

### Added

- **Mock server** (CRASH-002): JSON mappings serve canned responses honoring the configured HTTP method and status code. A request to a mapped path with the wrong method returns 405. Config is validated at load time, so a malformed mapping fails fast rather than serving silently.
- **Reverse proxy**: forwards a path to a configured upstream, passing the upstream status and body back verbatim, and returns 502 with a JSON error when the upstream call fails.
- **Fault injection** (CRASH-003): per-route base latency with jitter, and probabilistic error injection (rate plus status), on both mappings and proxies, unified in the `chaos` package.
- **Runtime fault toggle** (CRASH-003): `POST /admin/chaos` retunes a route's faults without a redeploy, so faults can be switched on during a demo.
- **Prometheus metrics** (CRASH-004): `/metrics` exposes `crashdummy_requests_total` (labeled route, method, status) and `crashdummy_request_duration_seconds` (labeled route, method), plus the default Go and process collectors.
- **Configuration via environment** (CRASH-005): `PORT` (default 10000) and `CRASHDUMMY_CONFIG_DIR` for the mappings/proxies/stubs root.
- **Production container** (CRASH-005): multi-stage build to a static, stripped binary on `gcr.io/distroless/static:nonroot`, running as a non-root user.
- **CI pipeline** (CRASH-007): golangci-lint, gosec, race-enabled tests, Docker build, trivy image scan (failing on HIGH/CRITICAL), and a GHCR publish on `v*` tags.
- **Unit tests** (CRASH-006): coverage of the `chaos`, `handlers`, and `metrics` packages, including the proxy failure path and the admin endpoint.
- **Documentation** (CRASH-009): a full configuration reference in the README (mappings, proxies, stubs, fault injection, the admin toggle, metrics, and environment variables).

### Changed

- **Modernized the recovered codebase** (CRASH-001): Go 1.25, `math/rand/v2`, `os` over the deprecated `ioutil`, real error handling (a failing upstream no longer nil-dereferences, and a 500 upstream no longer surfaces as 200), and removal of the hardcoded proxy branch and `InsecureSkipVerify`.

[Unreleased]: https://github.com/igorsilva-dev/crashdummy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/igorsilva-dev/crashdummy/releases/tag/v0.1.0

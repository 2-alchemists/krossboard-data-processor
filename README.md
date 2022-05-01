![Test and build](https://github.com/2-alchemists/krossboard-data-processor/workflows/test-and-build/badge.svg)
![Test and build](https://github.com/2-alchemists/krossboard-data-processor/workflows/golangci-lint/badge.svg)
![Build cloud images](https://github.com/2-alchemists/krossboard-data-processor/workflows/build-cloud-imagess/badge.svg)

---

# Overview
`krossboard-data-processor` is the backend component of [Krossboard](https://github.com/2-alchemists/krossboard).

`krossboard-data-processor` consists of the following capabilities:
 * Data collection from several Kubernetes source clusters.
 * Processing of cluster level usage analytics.
 * REST APIs to expose generated analytics (REST APIs). It's via these APIs that  [Krossboard UI](https://github.com/2-alchemists/krossboard-ui) consume to analytics data to produce its charts and dashboards.


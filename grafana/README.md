# Monitoring Demo

From [https://github.com/wick02/monitoring/](https://github.com/wick02/monitoring/).

## Pre reqs:

* Docker

## How to run:

Clone the repo and follow these steps

0. Create networks for docker

```
docker network create allservices
```

1. Start up the application

```
docker compose up --scale mimir=3
```

## Generate data

```
telemetrygen logs --logs=10 --workers=2 --otlp-http --otlp-insecure --otlp-endpoint=localhost:3100 --otlp-http-url-path=/otlp/v1/logs
telemetrygen metrics --metrics=2 --workers=2 --otlp-http --otlp-insecure --otlp-endpoint=localhost:9009 --otlp-http-url-path=/otlp/v1/metrics
```


## UI

When querying logs make sure to parse and format
```
{container="grafana-prometheus-1"} |= `` | logfmt | line_format `{{.msg}}`
```

# Monitoring Demo

Based on [https://github.com/wick02/monitoring/](https://github.com/wick02/monitoring/).

## Preparation:

Install `docker` and `docker-compose`.

Setup a docker network for services that need to communicate:
```
docker network create obslocal
```

Generate a password for Minio which will be used as storage.
```
echo "MINIO_ROOT_PASSWORD=$(openssl rand -base64 24)" >> .env
```
Docker Compose will read the `.env` file for us.


## How to run:

Launch the Grafana stack and an Open Telemetry Collector
```
docker compose up [-d]
```
and open [http://localhost:3000](http://localhost:3000).
The default login is admin/admin.

The Otel collector runs on the host network listening on port 4317 & 4318.


## Generate data

To test your setup, use the `telemetrygen` tool to generate some fake data:
```
go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest

# Logs
telemetrygen logs --logs=10 --workers=2 --otlp-http --otlp-insecure --otlp-endpoint=localhost:3100 --otlp-http-url-path=/otlp/v1/logs

# Metrics
telemetrygen metrics --metrics=2 --workers=2 --otlp-http --otlp-insecure --otlp-endpoint=localhost:3200 --otlp-http-url-path=/otlp/v1/metrics

# Traces
telemetrygen traces --traces=10 --workers=2 --otlp-http --otlp-insecure --otlp-endpoint=localhost:3302
```

Note that if you want to go through the Otel Collector, then send data to port `4318` instead, without any endpoint customization.


## Tips

Grafana can be stingy showing data when not searching for anything specific.
Use a query like
```
<label> =~ .+
```
to show all matches of _something_ reporting under the `<label>`.

When querying logs make sure to parse and format
```
{container="grafana-prometheus-1"} |= `` | logfmt | line_format `{{.msg}}`
```

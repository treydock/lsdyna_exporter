[![Build Status](https://circleci.com/gh/treydock/lsdyna_exporter/tree/master.svg?style=shield)](https://circleci.com/gh/treydock/lsdyna_exporter)
[![codecov](https://codecov.io/gh/treydock/lsdyna_exporter/branch/master/graph/badge.svg)](https://codecov.io/gh/treydock/lsdyna_exporter)
[![GitHub release](https://img.shields.io/github/v/release/treydock/lsdyna_exporter?include_prereleases&sort=semver)](https://github.com/treydock/lsdyna_exporter/releases/latest)
![GitHub All Releases](https://img.shields.io/github/downloads/treydock/lsdyna_exporter/total)
![Docker Pulls](https://img.shields.io/docker/pulls/treydock/lsdyna_exporter)

# ls-dyna Prometheus exporter

The ls-dyna exporter collects metrics from the ls-dyna license server.
The `/lsdyna` metrics endpoint exposes the license server metrics.
The `/metrics` endpoint exposes metrics about the exporter runtime.

## Usage

The only required flag is `--path.lstc_qrun`. This must point to the `lstc_qrun` binary capable to communicating with the ls-dyna license server.

This exporter is designed to run on a central host and communicate with remote ls-dyna license servers. It's possible to run locally on the ls-dyna license server but you will still need to provide the `target` query parameter.

Queries to the exporter would look like `http://localhost:9309/lsdyna?target=port@host` where `port` is the ls-dyna license server port and `host` is the license server host name.

## Prometheus configs

The following example assumes this exporter is running on the Prometheus server and communicating to a remote ls-dyna license server.

```yaml
- job_name: lsdyna
  metrics_path: /lsdyna
  static_configs:
  - targets:
    - 31011@license-host.example.com
  relabel_configs:
  - source_labels: [__address__]
    target_label: __param_target
  - source_labels: [__param_target]
    target_label: instance
  - target_label: __address__
    replacement: 127.0.0.1:9309
```

## Docker

Example of running the Docker container

```
docker run --rm -d -p 9309:9309 -v "/usr/local/bin/lstc_qrun:/lstc_qrun:ro" treydock/lsdyna_exporter --path.lstc_qrun=/lstc_qrun
```

## Install

Download the [latest release](https://github.com/treydock/lsdyna_exporter/releases)

Add the user that will run `lsdyna_exporter`

```
groupadd -r lsdyna_exporter
useradd -r -d /var/lib/lsdyna_exporter -s /sbin/nologin -M -g lsdyna_exporter -M lsdyna_exporter
```

Install compiled binaries after extracting tar.gz from release page.

```
cp /tmp/lsdyna_exporter /usr/local/bin/lsdyna_exporter
```

Add systemd unit file and start service. Modify the `ExecStart` with desired flags.

```
cp systemd/lsdyna_exporter.service /etc/systemd/system/lsdyna_exporter.service
systemctl daemon-reload
systemctl start lsdyna_exporter
```

## Build from source

To produce the `lsdyna_exporter` binary:

```
make build
```

Or

```
go get github.com/treydock/lsdyna_exporter
```

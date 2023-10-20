# PicoShare

[![CircleCI](https://circleci.com/gh/mtlynch/picoshare.svg?style=svg)](https://circleci.com/gh/mtlynch/picoshare)
[![Docker Version](https://img.shields.io/docker/v/mtlynch/picoshare?sort=semver&maxAge=86400)](https://hub.docker.com/r/mtlynch/picoshare/)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/m/mtlynch/picoshare)](https://github.com/mtlynch/picoshare/commits/master)
[![GitHub last commit](https://img.shields.io/github/last-commit/mtlynch/picoshare)](https://github.com/mtlynch/picoshare/commits/master)
[![Contributors](https://img.shields.io/github/contributors/mtlynch/picoshare)](https://github.com/mtlynch/picoshare/graphs/contributors)
[![License](http://img.shields.io/:license-agpl-blue.svg?style=flat-square)](LICENSE)

## Overview

PicoShare is a minimalist service that allows you to share files easily.

- [Live demo](https://demo.pico.rocks)

[![PicoShare demo](https://raw.githubusercontent.com/mtlynch/picoshare/master/docs/readme-assets/demo.gif)](https://raw.githubusercontent.com/mtlynch/picoshare/master/docs/readme-assets/demo-full.gif)

## Why PicoShare?

There are a million services for sharing files, but none of them are quite like PicoShare. Here are PicoShare's advantages:

- **Direct download links**: PicoShare gives you a direct download link you can share with anyone. They can view or download the file with no ads or signups.
- **No file restrictions**: Unlike sites like imgur, Vimeo, or SoundCloud that only allow you to share specific types of files, PicoShare lets you share any file of any size.
- **No resizing/re-encoding**: If you upload media like images, video, or audio, PicoShare never forces you to wait on re-encoding. You get a direct download link as soon as you upload the file, and PicoShare never resizes or re-encodes your file.

## Run PicoShare

### From source

```bash
PS_SHARED_SECRET=somesecretpass PORT=4001 \
  go run cmd/picoshare/main.go
```

### From Docker

To run PicoShare within a Docker container, mount a volume from your local system to store the PicoShare sqlite database.

```bash
docker run \
  --env "PORT=4001" \
  --env "PS_SHARED_SECRET=somesecretpass" \
  --publish 4001:4001/tcp \
  --volume "${PWD}/data:/data" \
  --name picoshare \
  mtlynch/picoshare
```

### From Docker + cloud data replication

If you specify settings for a [Litestream](https://litestream.io/)-compatible cloud storage location, PicoShare will automatically replicate your data.

You can kill the container and start it later, and PicoShare will restore your data from the cloud storage location and continue as if there was no interruption.

```bash
PORT=4001
PS_SHARED_SECRET="somesecretpass"
LITESTREAM_BUCKET=YOUR-LITESTREAM-BUCKET
LITESTREAM_ENDPOINT=YOUR-LITESTREAM-ENDPOINT
LITESTREAM_ACCESS_KEY_ID=YOUR-ACCESS-ID
LITESTREAM_SECRET_ACCESS_KEY=YOUR-SECRET-ACCESS-KEY

docker run \
  --publish "${PORT}:${PORT}/tcp" \
  --env "PORT=${PORT}" \
  --env "PS_SHARED_SECRET=${PS_SHARED_SECRET}" \
  --env "LITESTREAM_ACCESS_KEY_ID=${LITESTREAM_ACCESS_KEY_ID}" \
  --env "LITESTREAM_SECRET_ACCESS_KEY=${LITESTREAM_SECRET_ACCESS_KEY}" \
  --env "LITESTREAM_BUCKET=${LITESTREAM_BUCKET}" \
  --env "LITESTREAM_ENDPOINT=${LITESTREAM_ENDPOINT}" \
  --name picoshare \
  mtlynch/picoshare
```

Notes:

- Only run one Docker container for each Litestream location.
  - PicoShare can't sync writes across multiple instances.

### Using Docker Compose

To run PicoShare under docker-compose, copy the following to a file called `docker-compose.yml` and then run `docker-compose up`.

```yaml
version: "3.2"
services:
  picoshare:
    image: mtlynch/picoshare
    environment:
      - PORT=4001
      - PS_SHARED_SECRET=dummypass # Change to any password
    ports:
      - 4001:4001
    command: -db /data/store.db
    volumes:
      - ./data:/data
```

## Parameters

### Command-line flags

| Flag      | Meaning                                                                  | Default Value     |
| --------- | ------------------------------------------------------------------------ | ----------------- |
| `-db`     | Path to SQLite database                                                  | `"data/store.db"` |
| `-vacuum` | Vacuum database periodically to reclaim disk space (increases RAM usage) | `false`           |

### Environment variables

| Environment Variable | Meaning                                                                              |
| -------------------- | ------------------------------------------------------------------------------------ |
| `PORT`               | TCP port on which to listen for HTTP connections (defaults to 4001).                 |
| `PS_BEHIND_PROXY`    | Set to `"true"` for better logging when PicoShare is running behind a reverse proxy. |
| `PS_SHARED_SECRET`   | (required) Specifies a passphrase for the admin user to log in to PicoShare.         |

### Docker environment variables

You can adjust behavior of the Docker container by specifying these Docker-specific variables with `docker run -e`:

| Environment Variable           | Meaning                                                                                               |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- |
| `LITESTREAM_BUCKET`            | Litestream-compatible cloud storage bucket where Litestream should replicate data.                    |
| `LITESTREAM_ENDPOINT`          | Litestream-compatible cloud storage endpoint where Litestream should replicate data.                  |
| `LITESTREAM_ACCESS_KEY_ID`     | Litestream-compatible cloud storage access key ID to the bucket where you want to replicate data.     |
| `LITESTREAM_SECRET_ACCESS_KEY` | Litestream-compatible cloud storage secret access key to the bucket where you want to replicate data. |
| `LITESTREAM_RETENTION`         | The amount of time Litestream snapshots & WAL files will be kept (defaults to 72h).                   |

### Docker build args

If you rebuild the Docker image from source, you can adjust the build behavior with `docker build --build-arg`:

| Build Arg            | Meaning                                                                     | Default Value |
| -------------------- | --------------------------------------------------------------------------- | ------------- |
| `litestream_version` | Version of [Litestream](https://litestream.io/) to use for data replication | `0.3.9`       |

## Deployment

PicoShare is easy to deploy to cloud hosting platforms:

- [fly.io](docs/deployment/fly.io.md)
- [Elestio](https://elest.io/open-source/picoshare)

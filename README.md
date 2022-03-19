# picoshare

[![CircleCI](https://circleci.com/gh/mtlynch/picoshare.svg?style=svg)](https://circleci.com/gh/mtlynch/picoshare)
[![Docker Pulls](https://img.shields.io/docker/pulls/mtlynch/picoshare.svg?maxAge=86400)](https://hub.docker.com/r/mtlynch/picoshare/)
[![GitHub commit activity](https://img.shields.io/github/commit-activity/m/mtlynch/picoshare)](https://github.com/mtlynch/picoshare/commits/master)
[![GitHub last commit](https://img.shields.io/github/last-commit/mtlynch/picoshare)](https://github.com/mtlynch/picoshare/commits/master)
[![License](http://img.shields.io/:license-agpl-blue.svg?style=flat-square)](LICENSE)

A minimal, easy-to-host service for sharing images and other files.

## Experimental

PicoShare is still highly experimental and not yet ready for production usage.

You're welcome to try it, but expect frequent changes to the UI, database schema, and REST endpoints.

## Run PicoShare

### From source

```bash
PS_SHARED_SECRET=somesecretpass PORT=3001 \
  go run main.go
```

### From Docker

To run PicoShare within a Docker container, mount a volume from your local system to store the PicoShare sqlite database.

```bash
docker run \
  --env "PORT=3001" \
  --env "PS_SHARED_SECRET=somesecretpass" \
  --publish 3001:3001/tcp \
  --volume "${PWD}/data:/data" \
  --name picoshare \
  mtlynch/picoshare
```

### From Docker + cloud data replication

If you specify settings for a [Litestream](https://litestream.io/)-compatible cloud storage location, PicoShare will automatically replicate your data.

You can kill the container and start it later, and PicoShare will restore your data from the cloud storage location and continue as if there was no interruption.

```bash
LITESTREAM_BUCKET=YOUR-LITESTREAM-BUCKET
LITESTREAM_ENDPOINT=YOUR-LITESTREAM-ENDPOINT
LITESTREAM_ACCESS_KEY_ID=YOUR-ACCESS-ID
LITESTREAM_SECRET_ACCESS_KEY=YOUR-SECRET-ACCESS-KEY

docker run \
  --env "PORT=3001" \
  --env "LITESTREAM_ACCESS_KEY_ID=${LITESTREAM_ACCESS_KEY_ID}" \
  --env "LITESTREAM_SECRET_ACCESS_KEY=${LITESTREAM_SECRET_ACCESS_KEY}" \
  --env "LITESTREAM_BUCKET=${LITESTREAM_BUCKET}" \
  --env "LITESTREAM_ENDPOINT=${LITESTREAM_BUCKET}" \
  --publish 3001:3001/tcp \
  --name picoshare \
  mtlynch/picoshare
```

Notes:

- Only run one Docker container for each Litestream location.
  - PicoShare can't sync writes across multiple instances.

## Parameters

### Command-line flags

| Flag  | Meaning                 | Default Value     |
| ----- | ----------------------- | ----------------- |
| `-db` | Path to SQLite database | `"data/store.db"` |

### Docker environment variables

You can adjust behavior of the Docker container by passing these parameters with `docker run -e`:

| Environment Variable           | Meaning                                                                                               |
| ------------------------------ | ----------------------------------------------------------------------------------------------------- |
| `PORT`                         | TCP port on which to listen for HTTP connections (defaults to 3001).                                  |
| `LITESTREAM_BUCKET`            | Litestream-compatible cloud storage bucket where Litestream should replicate data.                    |
| `LITESTREAM_ENDPOINT`          | Litestream-compatible cloud storage endpoint where Litestream should replicate data.                  |
| `LITESTREAM_ACCESS_KEY_ID`     | Litestream-compatible cloud storage access key ID to the bucket where you want to replicate data.     |
| `LITESTREAM_SECRET_ACCESS_KEY` | Litestream-compatible cloud storage secret access key to the bucket where you want to replicate data. |

### Docker build args

If you rebuild the Docker image from source, you can adjust the build behavior with `docker build --build-arg`:

| Build Arg            | Meaning                                                                     | Default Value |
| -------------------- | --------------------------------------------------------------------------- | ------------- |
| `litestream_version` | Version of [Litestream](https://litestream.io/) to use for data replication | `0.3.8`       |

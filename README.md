# picoshare

[![CircleCI](https://circleci.com/gh/mtlynch/picoshare.svg?style=svg)](https://circleci.com/gh/mtlynch/picoshare)
[![Docker Pulls](https://img.shields.io/docker/pulls/mtlynch/picoshare.svg?maxAge=604800)](https://hub.docker.com/r/mtlynch/picoshare/)
[![License](http://img.shields.io/:license-agpl-blue.svg?style=flat-square)](LICENSE)

A minimal, easy-to-host service for sharing images and other files.

## Run

### From source

```bash
PORT=3001 go run main.go
```

### From Docker

```bash
docker run \
  -e "PORT=3001" \
  -e "PS_SHARED_SECRET=somesecretpass" \
  -p 3001:3001/tcp \
  --name picoshare \
  mtlynch/picoshare
```

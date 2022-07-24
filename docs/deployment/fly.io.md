# Deploy to fly.io

fly.io is a fantastic host for PicoShare. You can run up to three instances 24/7 for a month under their free tier, which includes free SSL certificates and 3 GB of persistent disk.

## Pre-requisites

You'll need:

- A fly.io account (with billing activated)
- The `fly` CLI [already installed](https://fly.io/docs/getting-started/installing-flyctl/) and authenticated on your machine

## Create your app

```bash
RANDOM_SUFFIX="$(head /dev/urandom | tr -dc 'a-z0-9' | head -c 6 ; echo '')"
APP_NAME="picoshare-${RANDOM_SUFFIX}"

curl -s -L https://raw.githubusercontent.com/mtlynch/picoshare/master/docs/deployment/fly-assets/make-fly-config | \
  bash /dev/stdin "${APP_NAME}"

fly apps create --name "${APP_NAME}"
```

## Create a persistent volume (optional)

You can add a persistent volume to store PicoShare's data across server deploys.

**Note**: You'll need either a persistent volume or Litestream replication ([below](#set-your-litestream-environment-variables-optional)) to prevent PicoShare from losing all data on every server redeploy.

```bash
VOLUME_NAME="pico_data"
SIZE_IN_GB=3 # This is the limit of fly.io's free tier as of 2022-02-19
REGION="iad"

fly volumes create "${VOLUME_NAME}" \
  --region "${REGION}" \
  --size "${SIZE_IN_GB}" && \
  {
  cat << EOF
[mounts]
source="${VOLUME_NAME}"
destination="/data"
EOF
  } >> fly.toml
```

## Set your Litestream environment variables (optional)

You can use [Litestream](https://litestream.io) to replicate PicoShare's data to cloud storage.

**Note**: You'll need either a persistent volume ([above](#create-a-persistent-volume-optional)) or Litestream replication to prevent PicoShare from losing all data on every server redeploy.

```bash
LITESTREAM_ACCESS_KEY_ID=YOUR-ACCESS-ID
LITESTREAM_SECRET_ACCESS_KEY=YOUR-SECRET-ACCESS-KEY

fly secrets set \
  "LITESTREAM_ACCESS_KEY_ID=${LITESTREAM_ACCESS_KEY_ID}" \
  "LITESTREAM_SECRET_ACCESS_KEY=${LITESTREAM_SECRET_ACCESS_KEY}"
```

In your `fly.toml` file, add the following lines in the `[env]` section:

```toml
[env]
  # ...
  LITESTREAM_BUCKET="YOUR-CLOUD-STORAGE-BUCKET"
  LITESTREAM_ENDPOINT="YOUR-CLOUD-STORAGE-ENDPOINT"
```

For example, if you had a bucket called `my-pico-bucket` on Backblaze B2's `us-west-002` region, your config would look like this:

```toml
LITESTREAM_BUCKET="my-pico-bucket"
LITESTREAM_ENDPOINT="s3.us-west-002.backblazeb2.com"
```

## Set a passphrase

Choose a passphrase to secure your instance.

```bash
flyctl secrets set PS_SHARED_SECRET="somesecretpassphrase"
```

## Deploy

Finally, it's time to deploy your app.

```bash
# Change this to the latest Docker image tag
PICOSHARE_IMAGE="mtlynch/picoshare:1.0.0"

fly deploy \
  --region="${REGION}" \
  --image "${PICOSHARE_IMAGE}" && \
  PICOSHARE_URL="https://${APP_NAME}.fly.dev/" && \
  echo "Your PicoShare instance is now ready at: ${PICOSHARE_URL}"
```

## Increase RAM (optional)

By default, fly VMs include 256 MB of RAM. Your instance will need at least 512 MB of RAM to support large uploads.

```bash
fly scale memory 512
```

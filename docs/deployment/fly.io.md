# Deploy to fly.io

TODO(mtlynch): Fill out these instructions

## Create fly.toml

TODO(mtlynch): Explain how to create `fly.toml` file.

## Create your app

```bash
RANDOM_SUFFIX="$(head /dev/urandom | tr -dc 'a-z0-9' | head -c 6 ; echo '')"
APP_NAME="picoshare-${RANDOM_SUFFIX}"

fly apps create --name "${APP_NAME}"
```

## Create a persistent volume (optional)

```bash
VOLUME_NAME="pico_data"
SIZE_IN_GB=3 # This is the limit of fly.io's free tier as of 2022-02-19
REGION="iad"

fly volumes create "${VOLUME_NAME}" \
  --region "${REGION}" \
  --size "${SIZE_IN_GB}"
```

## Set your Litestream environment variables (optional)

You only need this if you're syncing data to a Litestream replica.

```bash
LITESTREAM_ACCESS_KEY_ID=YOUR-ACCESS-ID
LITESTREAM_SECRET_ACCESS_KEY=YOUR-SECRET-ACCESS-KEY

fly secrets set \
  "LITESTREAM_ACCESS_KEY_ID=${LITESTREAM_ACCESS_KEY_ID}" \
  "LITESTREAM_SECRET_ACCESS_KEY=${LITESTREAM_SECRET_ACCESS_KEY}"
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
PICOSHARE_IMAGE="mtlynch/picoshare:0.1.0"

fly deploy \
  --region="${REGION}" \
  --image "${PICOSHARE_IMAGE}" && \
  PICOSHARE_URL="https://${APP_NAME}.fly.dev/" && \
  echo "Your PicoShare instance is now ready at: ${PICOSHARE_URL}"
```

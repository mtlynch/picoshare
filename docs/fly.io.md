# Deploy to fly.io

TODO(mtlynch): Fill out these instructions

## Create a persistent volume (optional)

```bash
VOLUME_NAME="pico_data"
SIZE_IN_GB=3 # This is the limit of fly.io's free tier as of 2022-02-19
REGION="iad"

fly volumes create "${VOLUME_NAME}" \
  --encrypted \
  --region="${REGION}" \
  --size "${SIZE_IN_GB}"
```

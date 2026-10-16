# End-to-end test fixtures

`tiny-video.mp4` is a two-second, 64 by 64 pixel VP9 video used to verify native video playback in Chromium. Generate it with:

```bash
ffmpeg -f lavfi -i "testsrc2=size=64x64:rate=10" -t 2 -an \
  -c:v libvpx-vp9 -pix_fmt yuv420p -movflags +faststart \
  e2e/fixtures/tiny-video.mp4
```

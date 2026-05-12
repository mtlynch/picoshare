export function makeShortLink(fileId) {
  return `${window.location.origin}/-${fileId}`;
}

export function makeVerboseLink(fileId, filename) {
  return makeShortLink(fileId) + "/" + encodeURIComponent(filename);
}

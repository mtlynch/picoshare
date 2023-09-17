export function parseRfc3339(raw) {
  // Make sure the timestamp is valid.
  const timestamp = Date.parse(raw);
  if (isNaN(timestamp)) {
    return null;
  }
  return new Date(timestamp);
}

// Formats a Date object in semi-RFC3339 format with extra spacing and the hours
// offset in local time.
export function formatRfc3339Local(ts) {
  const year = ts.getFullYear();
  // Month is zero-based, so add 1.
  const month = (ts.getMonth() + 1).toString().padStart(2, "0");
  const day = ts.getDate().toString().padStart(2, "0");
  const hours = ts.getHours().toString().padStart(2, "0");
  const minutes = ts.getMinutes().toString().padStart(2, "0");
  const seconds = ts.getSeconds().toString().padStart(2, "0");

  const timezoneOffsetHours = Math.floor(ts.getTimezoneOffset() / 60);
  const timezoneOffsetMinutes = Math.abs(ts.getTimezoneOffset() % 60);

  const timezoneOffsetString =
    " " +
    (timezoneOffsetHours >= 0 ? "+" : "-") +
    timezoneOffsetHours.toString().padStart(2, "0") +
    timezoneOffsetMinutes.toString().padStart(2, "0");

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}${timezoneOffsetString}`;
}

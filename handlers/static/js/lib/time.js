export function parseRfc3339(raw) {
  // Make sure the timestamp is valid.
  const timestamp = Date.parse(raw);
  if (isNaN(timestamp)) {
    return null;
  }
  return new Date(timestamp);
}

export function formatDateInputValue(date) {
  const year = String(date.getFullYear()).padStart(4, "0");
  const month = zeroPad(date.getMonth() + 1, 2);
  const day = zeroPad(date.getDate(), 2);
  return `${year}-${month}-${day}`;
}

export function convertDateInputValueToRfc3339(raw) {
  const [year, month, day] = raw.split("-").map(Number);
  const date = new Date(0);
  date.setFullYear(year, month - 1, day);
  date.setHours(0, 0, 0, 0);
  return date.toISOString();
}

// Formats a Date object in semi-RFC3339 format with extra spacing and the hours
// offset in local time.
export function formatRfc3339Local(ts) {
  const year = ts.getFullYear();
  // Month is zero-based, so add 1.
  const month = zeroPad(ts.getMonth() + 1, 2);
  const day = zeroPad(ts.getDate(), 2);
  const hours = zeroPad(ts.getHours(), 2);
  const minutes = zeroPad(ts.getMinutes(), 2);
  const seconds = zeroPad(ts.getSeconds(), 2);

  const timezoneOffsetHours = Math.floor(ts.getTimezoneOffset() / 60);
  const timezoneOffsetMinutes = Math.abs(ts.getTimezoneOffset() % 60);

  const timezoneOffsetString =
    " " +
    (timezoneOffsetHours >= 0 ? "+" : "-") +
    zeroPad(timezoneOffsetHours, 2) +
    zeroPad(timezoneOffsetMinutes, 2);

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}${timezoneOffsetString}`;
}

function zeroPad(n, padCount) {
  return n.toString().padStart(padCount, "0");
}

export function readDbTokenCookie(cookies) {
  for (const cookie of cookies) {
    if (cookie.name === "db-token") {
      return cookie;
    }
  }
  return undefined;
}

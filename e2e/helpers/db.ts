interface Cookie {
  name: string;
  value: string;
  domain?: string;
  path?: string;
  expires?: number;
  httpOnly?: boolean;
  secure?: boolean;
  sameSite?: "Strict" | "Lax" | "None";
}

export function readDbTokenCookie(cookies: Cookie[]): Cookie | undefined {
  for (const cookie of cookies) {
    if (cookie.name === "db-token") {
      return cookie;
    }
  }
  return undefined;
}

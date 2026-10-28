import { test, expect } from "./fixtures";

test.describe.configure({ mode: "parallel" });

test("accepts a normal Host header", async ({ request }) => {
  const response = await request.get("/", {
    headers: { Host: "files.example.com" },
  });

  expect(response.status()).toBe(200);
});

test("rejects an oversized Host and continues serving requests", async ({
  request,
}) => {
  const oversizedResponse = await request.get("/", {
    headers: { Host: `${"a".repeat(32 * 1024)}.example.com` },
  });

  expect(oversizedResponse.status()).toBe(431);

  const normalResponse = await request.get("/");

  expect(normalResponse.status()).toBe(200);
});

test("rejects oversized combined request headers", async ({ request }) => {
  const response = await request.get("/", {
    headers: {
      "X-First": "a".repeat(4 * 1024),
      "X-Second": "b".repeat(4 * 1024),
      "X-Third": "c".repeat(4 * 1024),
      "X-Fourth": "d".repeat(4 * 1024),
    },
  });

  expect(response.status()).toBe(431);
});

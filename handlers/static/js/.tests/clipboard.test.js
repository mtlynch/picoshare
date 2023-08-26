import { describe, it } from "mocha";
import assert from "assert";
import { sortClipboardItems } from "../lib/clipboard.js";

describe("sortClipboardItems", () => {
  it("prioritizes images", () => {
    assert.deepEqual(
      [
        {
          kind: "file",
          type: "image/png",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ],
      sortClipboardItems([
        {
          kind: "string",
          type: "text/html",
        },
        {
          kind: "file",
          type: "image/png",
        },
      ])
    );

    assert.deepEqual(
      [
        {
          kind: "file",
          type: "image/png",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ],
      sortClipboardItems([
        {
          kind: "file",
          type: "image/png",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ])
    );
  });

  it("prioritizes plaintext over other text types", () => {
    assert.deepEqual(
      [
        {
          kind: "string",
          type: "text/plain",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ],
      sortClipboardItems([
        {
          kind: "string",
          type: "text/plain",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ])
    );

    assert.deepEqual(
      [
        {
          kind: "string",
          type: "text/plain",
        },
        {
          kind: "string",
          type: "text/html",
        },
      ],
      sortClipboardItems([
        {
          kind: "string",
          type: "text/html",
        },
        {
          kind: "string",
          type: "text/plain",
        },
      ])
    );
  });
});

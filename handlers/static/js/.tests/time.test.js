import assert from "assert";
import { describe, it } from "mocha";
import {
  convertDateInputValueToRfc3339,
  formatDateInputValue,
} from "../lib/time.js";

describe("formatDateInputValue", () => {
  it("formats a local calendar date", () => {
    const date = new Date(2029, 8, 3, 13, 45, 0);

    assert.equal(formatDateInputValue(date), "2029-09-03");
  });
});

describe("convertDateInputValueToRfc3339", () => {
  it("converts a calendar date from local midnight", () => {
    const date = new Date(0);
    date.setFullYear(2029, 8, 3);
    date.setHours(0, 0, 0, 0);

    assert.equal(
      convertDateInputValueToRfc3339("2029-09-03"),
      date.toISOString(),
    );
  });
});

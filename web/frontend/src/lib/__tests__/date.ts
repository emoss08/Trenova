import { expect, test } from "vitest";
import { formatTimestamp } from "../date";

test("Parses date to human readable format", () => {
  const timestamp = new Date().toISOString();
  const formattedDate = formatTimestamp(timestamp);

  expect(formattedDate).toBe("0 secs ago");
});

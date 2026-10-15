import { describe, expect, it } from "vitest";
import { shortDigest } from "../digest";

describe("shortDigest", () => {
  it("keeps the algorithm and the first twelve characters of the sum", () => {
    expect(shortDigest("sha256:5f1c9a0b7e3d2c1f00aa11bb22cc33dd")).toBe("sha256:5f1c9a0b7e3d");
  });

  it("keeps a sum shorter than twelve characters whole", () => {
    expect(shortDigest("sha256:5f1c")).toBe("sha256:5f1c");
  });

  it("shortens a bare sum and reads nothing as nothing", () => {
    expect(shortDigest("5f1c9a0b7e3d2c1f")).toBe("5f1c9a0b7e3d");
    expect(shortDigest("")).toBe("");
  });
});

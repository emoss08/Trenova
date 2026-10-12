import { describe, expect, it } from "vitest";
import { artifactRefId } from "../artifact-ref";

describe("artifactRefId", () => {
  it("reads the id an artifact link names", () => {
    expect(artifactRefId("artifact:art_01M4P4440MSW18YW2GEDZP5WQ3")).toBe(
      "art_01M4P4440MSW18YW2GEDZP5WQ3",
    );
  });

  it("is null for any other link", () => {
    expect(artifactRefId("https://example.com/artifact:art_1")).toBeNull();
    expect(artifactRefId("/shipments/shp_1")).toBeNull();
    expect(artifactRefId(undefined)).toBeNull();
  });

  it("is null for an address that is not an id", () => {
    expect(artifactRefId("artifact:")).toBeNull();
    expect(artifactRefId("artifact:art_1)<script>")).toBeNull();
  });
});

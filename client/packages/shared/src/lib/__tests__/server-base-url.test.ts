import { describe, expect, it } from "vitest";
import { serverBaseUrl } from "../api-url";

const PAGE = "https://tms.acme.test/capture/devices?tab=install";

describe("serverBaseUrl", () => {
  it("drops the versioned prefix from an API on its own origin", () => {
    expect(serverBaseUrl(PAGE, "http://localhost:8080/api/v1")).toBe("http://localhost:8080");
  });

  it("resolves a relative API base against the page", () => {
    expect(serverBaseUrl(PAGE, "/api/v1")).toBe("https://tms.acme.test");
  });

  it("keeps the path the server is mounted under", () => {
    expect(serverBaseUrl(PAGE, "https://acme.test/tms/api/v1/")).toBe("https://acme.test/tms");
  });

  it("leaves a base without the versioned prefix whole, without a trailing slash", () => {
    expect(serverBaseUrl(PAGE, "https://api.acme.test/")).toBe("https://api.acme.test");
  });

  it("does not carry the page's query or fragment into the address", () => {
    expect(serverBaseUrl(`${PAGE}#install`, "/api/v1")).toBe("https://tms.acme.test");
  });
});

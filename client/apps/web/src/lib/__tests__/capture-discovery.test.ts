import { describe, expect, it } from "vitest";
import { captureApiBaseUrl, captureWellKnownBody } from "../../../vite/capture-discovery";

describe("captureApiBaseUrl", () => {
  it("takes the versioned prefix off an absolute API base", () => {
    expect(captureApiBaseUrl("https://api.trenova.app/api/v1")).toBe("https://api.trenova.app");
    expect(captureApiBaseUrl("https://api.trenova.app/api/v1/")).toBe("https://api.trenova.app");
    expect(captureApiBaseUrl("https://acme.example/tms/api/v1")).toBe("https://acme.example/tms");
  });

  it("publishes a relative base for an API served beside the app", () => {
    expect(captureApiBaseUrl("/api/v1")).toBe("/");
    expect(captureApiBaseUrl("/tms/api/v1")).toBe("/tms");
    expect(captureApiBaseUrl("")).toBe("/");
  });

  it("writes one stable line naming the product", () => {
    expect(captureWellKnownBody("https://api.trenova.app/api/v1")).toBe(
      '{"product":"trenova","apiBaseUrl":"https://api.trenova.app"}\n',
    );
  });
});

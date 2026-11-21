import { serverBaseUrl } from "@trenova/shared/lib/api-url";
import { describe, expect, it } from "vitest";
import { scimBaseUrl } from "../_components/provisioning/constants";

describe("scimBaseUrl", () => {
  it("is the server's address followed by the directory's SCIM 2.0 path", () => {
    expect(scimBaseUrl("acme")).toBe(`${serverBaseUrl()}/api/v1/scim/acme/v2`);
  });

  it("keeps a tenant slug from changing the path", () => {
    expect(scimBaseUrl("acme/v2/Users")).toBe(
      `${serverBaseUrl()}/api/v1/scim/acme%2Fv2%2FUsers/v2`,
    );
  });
});

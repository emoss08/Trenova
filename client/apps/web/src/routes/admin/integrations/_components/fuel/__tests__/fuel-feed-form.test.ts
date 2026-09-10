import type { ConfigFieldSpec } from "@/types/integration";
import { describe, expect, it } from "vitest";
import { isFieldVisible } from "../fuel-feed-form";

function spec(key: string, overrides: Partial<ConfigFieldSpec> = {}): ConfigFieldSpec {
  return {
    key,
    label: key,
    type: "string",
    required: false,
    sensitive: false,
    ...overrides,
  };
}

describe("fuel feed field visibility", () => {
  // The record layout only means anything for a fixed-width file. Showing it for
  // a CSV connection invites somebody to fill it in and wonder why it is ignored.
  it("shows the record layout only for a fixed-width file", () => {
    const layout = spec("fixedWidthLayout");

    expect(isFieldVisible(layout, { fileFormat: "FixedWidth", hasFormatField: true })).toBe(true);
    expect(isFieldVisible(layout, { fileFormat: "Delimited", hasFormatField: true })).toBe(false);
  });

  // Ramp has no file format at all, so the layout field must not appear there
  // even though the key is in the shared set.
  it("hides the record layout on a connection with no file format", () => {
    expect(
      isFieldVisible(spec("fixedWidthLayout"), { fileFormat: "FixedWidth", hasFormatField: false }),
    ).toBe(false);
  });

  it("shows only the credential the chosen authentication uses", () => {
    const password = spec("password", { sensitive: true, type: "password" });
    const privateKey = spec("privateKey", { sensitive: true, type: "password" });

    expect(isFieldVisible(password, { authMode: "password", hasFormatField: true })).toBe(true);
    expect(isFieldVisible(privateKey, { authMode: "password", hasFormatField: true })).toBe(false);

    expect(isFieldVisible(password, { authMode: "privateKey", hasFormatField: true })).toBe(false);
    expect(isFieldVisible(privateKey, { authMode: "privateKey", hasFormatField: true })).toBe(true);
  });

  // Authentication defaults to password, so an unset value must not hide both
  // credentials and leave the form impossible to complete.
  it("defaults to the password credential when authentication is unset", () => {
    expect(isFieldVisible(spec("password"), { hasFormatField: true })).toBe(true);
    expect(isFieldVisible(spec("privateKey"), { hasFormatField: true })).toBe(false);
  });

  it("shows every other field regardless of the mode", () => {
    for (const key of ["host", "port", "username", "remoteDirectory", "accountNumber"]) {
      expect(isFieldVisible(spec(key), { hasFormatField: true })).toBe(true);
    }
  });
});

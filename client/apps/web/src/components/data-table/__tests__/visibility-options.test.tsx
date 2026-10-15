import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useVisibilityOptions } from "../data-table-save-config-dialog";

describe("useVisibilityOptions", () => {
  it("offers sharing with the organization only to a role that may share views", () => {
    const shared = renderHook(() => useVisibilityOptions(true)).result.current;
    const privateOnly = renderHook(() => useVisibilityOptions(false)).result.current;

    expect(shared.map((option) => option.value)).toEqual(["Private", "Public"]);
    expect(privateOnly.map((option) => option.value)).toEqual(["Private"]);
  });
});

import { afterEach, describe, expect, it, vi } from "vitest";
import { preferenceStorage } from "../preference-storage";

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("preferenceStorage", () => {
  it("reads back what it wrote, and nothing for a key it never wrote", () => {
    preferenceStorage.setItem("resizable:lading-studio:files", '{"files":24,"work":76}');
    expect(preferenceStorage.getItem("resizable:lading-studio:files")).toBe(
      '{"files":24,"work":76}',
    );
    expect(preferenceStorage.getItem("resizable:unknown")).toBeNull();
  });

  it("reads nothing, without throwing, when the browser refuses storage", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("The operation is insecure.", "SecurityError");
    });
    expect(preferenceStorage.getItem("resizable:lading-studio:files")).toBeNull();
  });

  it("drops a write, without throwing, when storage is full", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("The quota has been exceeded.", "QuotaExceededError");
    });
    expect(() => preferenceStorage.setItem("resizable:lading-studio:files", "{}")).not.toThrow();
  });
});

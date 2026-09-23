import { describe, expect, it } from "vitest";
import { conversationAtPath, conversationPath, isDeskPath } from "../conversation-path";

describe("conversationPath", () => {
  it("opens a conversation at the Desk", () => {
    expect(conversationPath("athr_01J")).toBe("/desk/t/athr_01J");
  });

  it("keeps an id that is not a plain token inside its segment", () => {
    expect(conversationPath("a/b?c")).toBe("/desk/t/a%2Fb%3Fc");
  });
});

describe("conversationAtPath", () => {
  it("reads the conversation back from its path", () => {
    expect(conversationAtPath(conversationPath("athr_01J"))).toBe("athr_01J");
    expect(conversationAtPath(conversationPath("a/b?c"))).toBe("a/b?c");
  });

  it("tolerates a trailing slash", () => {
    expect(conversationAtPath("/desk/t/athr_1/")).toBe("athr_1");
  });

  it.each(["/desk", "/desk/t/", "/desk/decisions", "/desk/t/athr_1/extra", "/detention/desk"])(
    "is null for %s",
    (pathname) => {
      expect(conversationAtPath(pathname)).toBeNull();
    },
  );

  it("is null for a malformed escape rather than throwing", () => {
    expect(conversationAtPath("/desk/t/%E0%A4%A")).toBeNull();
  });
});

describe("isDeskPath", () => {
  it.each(["/desk", "/desk/", "/desk/t/athr_1", "/desk/watchtower"])("is true for %s", (path) => {
    expect(isDeskPath(path)).toBe(true);
  });

  it.each(["/", "/detention/desk", "/desktop", "/shipment-management/shipments"])(
    "is false for %s",
    (path) => {
      expect(isDeskPath(path)).toBe(false);
    },
  );
});

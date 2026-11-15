import { describe, expect, it } from "vitest";
import {
  MAX_RETURN_PATH_LENGTH,
  returnPathFor,
  returnPathFrom,
  safeReturnPath,
  type SignInPaths,
} from "../return-path";

const web: SignInPaths = { loginPath: "/login", home: "/" };
const dash: SignInPaths = { loginPath: "/dash/login", home: "/dash" };

describe("safeReturnPath", () => {
  it("follows a path on this origin with its query", () => {
    expect(safeReturnPath("/billing/invoices?status=open", web)).toBe(
      "/billing/invoices?status=open",
    );
    expect(safeReturnPath("/", web)).toBe("/");
  });

  it("returns the path as it was judged, normalized and without a fragment", () => {
    expect(safeReturnPath("/billing/./invoices/../invoices#row-4", web)).toBe("/billing/invoices");
  });

  it.each([
    ["nothing", null],
    ["nothing", undefined],
    ["an empty value", ""],
    ["a bare word", "billing"],
    ["an absolute https address", "https://evil.com/billing"],
    ["an absolute address on this host's name", "http://localhost/billing"],
    ["a javascript: address", "javascript:alert(1)"],
    ["a data: address", "data:text/html,<script>alert(1)</script>"],
    ["a protocol-relative address", "//evil.com"],
    ["a protocol-relative address with a path", "//evil.com/billing"],
    ["a triple slash", "///evil.com"],
    ["a backslash after the slash", "/\\evil.com"],
    ["a backslash anywhere in the path", "/billing\\..\\..\\evil.com"],
    ["a doubled backslash", "\\\\evil.com"],
    ["an encoded slash pair", "/%2F%2Fevil.com"],
    ["an encoded slash pair in lower case", "/%2f%2fevil.com"],
    ["a double-encoded slash pair", "/%252F%252Fevil.com"],
    ["an encoded backslash", "/%5Cevil.com"],
    ["an encoded backslash pair", "/%5C%5Cevil.com"],
    ["dot segments that climb to a protocol-relative path", "/billing/..//evil.com"],
    ["a tab", "/\tbilling"],
    ["a newline", "/billing\n/evil"],
    ["an encoded newline", "/billing%0A"],
    ["an encoded null", "/billing%00"],
    ["a malformed escape", "/billing%E0%A4%A"],
    ["a scheme behind a leading space", " javascript:alert(1)"],
  ])("refuses %s", (_, value) => {
    expect(safeReturnPath(value, web)).toBeNull();
  });

  it("refuses a path longer than the cap and follows one at it", () => {
    const atCap = `/${"a".repeat(MAX_RETURN_PATH_LENGTH - 1)}`;

    expect(safeReturnPath(atCap, web)).toBe(atCap);
    expect(safeReturnPath(`${atCap}a`, web)).toBeNull();
  });

  it.each([
    ["the sign-in page", "/login"],
    ["the sign-in page with a query", "/login?redirect=%2Fbilling"],
    ["the sign-in page with a trailing slash", "/login/"],
    ["the sign-in page in another case", "/LOGIN"],
    ["a tenant's sign-in page", "/login/acme"],
    ["the sign-in page reached by dot segments", "/billing/../login"],
    ["the sign-in page behind an encoded letter", "/%6Cogin"],
  ])("refuses %s, so a sign-in never returns to it", (_, value) => {
    expect(safeReturnPath(value, web)).toBeNull();
  });

  it("follows a path that only starts with the sign-in page's letters", () => {
    expect(safeReturnPath("/login-history", web)).toBe("/login-history");
  });

  it("keeps an app with its own home inside it", () => {
    expect(safeReturnPath("/dash/loads/asg_1?tab=stops", dash)).toBe("/dash/loads/asg_1?tab=stops");
    expect(safeReturnPath("/dash", dash)).toBe("/dash");
    expect(safeReturnPath("/billing/invoices", dash)).toBeNull();
    expect(safeReturnPath("/dashboard", dash)).toBeNull();
    expect(safeReturnPath("/dash/login", dash)).toBeNull();
  });
});

describe("returnPathFor", () => {
  it("carries the address the person was going to", () => {
    expect(returnPathFor("/billing/invoices?status=open", web)).toBe(
      "/billing/invoices?status=open",
    );
  });

  it("carries nothing for the app's home, which a sign-in reaches anyway", () => {
    expect(returnPathFor("/", web)).toBeNull();
    expect(returnPathFor("/dash", dash)).toBeNull();
  });

  it("carries home when it holds a query", () => {
    expect(returnPathFor("/?assistant=open", web)).toBe("/?assistant=open");
  });

  it("carries nothing unsafe", () => {
    expect(returnPathFor("//evil.com", web)).toBeNull();
    expect(returnPathFor("/login", web)).toBeNull();
  });
});

describe("returnPathFrom", () => {
  it("follows the safe return path the sign-in page was opened with", () => {
    expect(
      returnPathFrom(new URLSearchParams("redirect=%2Fbilling%2Finvoices%3Fstatus%3Dopen"), web),
    ).toBe("/billing/invoices?status=open");
  });

  it("goes home when the page carries none", () => {
    expect(returnPathFrom(new URLSearchParams(""), web)).toBe("/");
    expect(returnPathFrom(new URLSearchParams("signed_out=expired"), dash)).toBe("/dash");
  });

  it("goes home instead of following an unsafe one", () => {
    expect(returnPathFrom(new URLSearchParams("redirect=https%3A%2F%2Fevil.com"), web)).toBe("/");
    expect(returnPathFrom(new URLSearchParams("redirect=%2F%2Fevil.com"), web)).toBe("/");
    expect(returnPathFrom(new URLSearchParams("redirect=%2Fbilling"), dash)).toBe("/dash");
  });
});

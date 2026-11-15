import { describe, expect, it } from "vitest";
import { isAppPage } from "@/lib/link-foresight";

const at = (path: string) => new URL(path, window.location.origin);

describe("isAppPage", () => {
  it.each(["/", "/shipments", "/shipments/shp_1?tab=stops", "/desk/v"])(
    "treats %s as a page of the app",
    (path) => {
      expect(isAppPage(at(path))).toBe(true);
    },
  );

  it.each(["/api/v1/shipments", "/graphql", "/dash/", "/dash/loads", "/.well-known/x", "/logo.svg"])(
    "does not treat %s as a page of the app",
    (path) => {
      expect(isAppPage(at(path))).toBe(false);
    },
  );
});

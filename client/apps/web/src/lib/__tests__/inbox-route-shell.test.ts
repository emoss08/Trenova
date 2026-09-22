import { routes } from "@/router";
import { AppLayout } from "@/routes/app-layout";
import { isValidElement } from "react";
import type { RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function layoutsAbove(
  entries: RouteObject[],
  path: string,
  trail: unknown[] = [],
): unknown[] | null {
  for (const entry of entries) {
    const next = isValidElement(entry.element) ? [...trail, entry.element.type] : trail;
    if (entry.path === path) return next;
    if (entry.children) {
      const found = layoutsAbove(entry.children, path, next);
      if (found) return found;
    }
  }
  return null;
}

/**
 * The inbox was mounted beside the Desk, outside the app frame, and it has no
 * frame of its own: someone who opened it had no sidebar, no header and no way
 * to anywhere else. It is a working page, so it wears the app's chrome.
 */
describe("inbox route", () => {
  it("renders inside the app layout, so the sidebar is the way back", () => {
    const layouts = layoutsAbove(routes, "/inbox");

    expect(layouts).not.toBeNull();
    expect(layouts).toContain(AppLayout);
  });
});

import { routes } from "@/router";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { isValidElement } from "react";
import type { RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function routeAt(entries: RouteObject[], path: string): RouteObject | null {
  for (const entry of entries) {
    if (entry.path === path) return entry;
    const found = entry.children ? routeAt(entry.children, path) : null;
    if (found) return found;
  }
  return null;
}

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
 * Triage is worked in the Desk's room, beside the rest of what needs a person,
 * and that is the only place it lives.
 */
describe("triage route", () => {
  it("renders inside the Desk's shell", () => {
    const layouts = layoutsAbove(routes, "triage");

    expect(layouts).not.toBeNull();
    expect(layouts).toContain(DeskShellLayout);
  });

  it("has one address, with none kept beside it", () => {
    expect(routeAt(routes, "/triage")).toBeNull();
    expect(routeAt(routes, "/inbox")).toBeNull();
  });
});

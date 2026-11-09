import { routes } from "@/router";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { isValidElement } from "react";
import type { LoaderFunctionArgs, RouteObject } from "react-router";
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

async function redirectFrom(path: string, url: string): Promise<string | null> {
  const loader = routeAt(routes, path)?.loader;
  if (typeof loader !== "function") return null;
  const args: LoaderFunctionArgs = {
    request: new Request(url),
    url: new URL(url),
    pattern: path,
    params: {},
    context: undefined as unknown as LoaderFunctionArgs["context"],
  };
  const response = (await loader(args)) as Response;
  return response.headers.get("Location");
}

/**
 * Triage is worked in the Desk's room, beside the rest of what needs a person.
 * The addresses people saved for it, and the inbox's before it, keep working.
 */
describe("triage route", () => {
  it("renders inside the Desk's shell", () => {
    const layouts = layoutsAbove(routes, "triage");

    expect(layouts).not.toBeNull();
    expect(layouts).toContain(DeskShellLayout);
  });

  it("sends the old Triage address to the Desk, keeping the open message", async () => {
    await expect(redirectFrom("/triage", "http://localhost/triage?message=msg_1")).resolves.toBe(
      "/desk/triage?message=msg_1",
    );
  });

  it("sends the inbox address to the Desk too", async () => {
    await expect(redirectFrom("/inbox", "http://localhost/inbox?folder=waiting")).resolves.toBe(
      "/desk/triage?folder=waiting",
    );
  });
});

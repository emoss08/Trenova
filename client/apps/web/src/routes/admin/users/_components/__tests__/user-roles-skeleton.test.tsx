import { cleanup, render, screen, within } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it } from "vitest";
import { UserRolesSkeleton } from "../user-roles-skeleton";

afterEach(cleanup);

const hidden = { hidden: true } as const;

function renderAt(searchParams: string) {
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <UserRolesSkeleton />
    </NuqsTestingAdapter>,
  );
}

function headerCount() {
  const root = screen.getByLabelText("Loading users and roles");
  return within(root).getAllByRole("columnheader", hidden).length;
}

describe("UserRolesSkeleton", () => {
  // The first paint is the tab strip with the named tab's table under it, so
  // the table that loads lands in the outline's columns.
  it("draws the users table, with its selection column, by default", () => {
    renderAt("");

    expect(screen.getByLabelText("Loading users and roles")).toHaveAttribute("aria-busy", "true");
    expect(headerCount()).toBe(6);
  });

  it("draws the roles table when the address names the roles tab", () => {
    renderAt("?tab=roles");

    expect(headerCount()).toBe(4);
  });

  it("keeps its stand-ins out of the accessibility tree and carries no text", () => {
    renderAt("");

    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.getByLabelText("Loading users and roles")).toHaveTextContent("");
  });
});

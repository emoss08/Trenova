import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CardPagedList } from "../card-paged-list";
import type { PagedRows } from "../paged-rows";

function list(overrides: Partial<PagedRows<string>> = {}): PagedRows<string> {
  return {
    rows: ["one", "two"],
    total: 5,
    hasMore: true,
    loadingMore: false,
    loadMore: vi.fn(),
    pending: false,
    error: null,
    ...overrides,
  };
}

function draw(paged: PagedRows<string>) {
  return render(
    <CardPagedList
      list={paged}
      emptyText="Nothing here yet."
      errorText="The rows could not be read."
      nextErrorText="The next rows could not be read."
      loadingLabel="Reading more rows…"
    >
      {(rows) => (
        <ul>
          {rows.map((row) => (
            <li key={row}>{row}</li>
          ))}
        </ul>
      )}
    </CardPagedList>,
  );
}

describe("CardPagedList", () => {
  it("draws the rows with how many are shown of how many, and reads the next page on request", () => {
    const paged = list();
    draw(paged);

    expect(screen.getAllByRole("listitem").map((item) => item.textContent)).toEqual(["one", "two"]);
    expect(screen.getByText("2 of 5 shown")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show more" }));
    expect(paged.loadMore).toHaveBeenCalled();
  });

  it("says it is reading the next page, and leaves the button off once there is none", () => {
    const { rerender } = draw(list({ loadingMore: true }));
    expect(screen.getByRole("button", { name: "Reading more rows…" })).toBeDisabled();

    rerender(
      <CardPagedList
        list={list({ hasMore: false, total: null })}
        emptyText="Nothing here yet."
        errorText="The rows could not be read."
        nextErrorText="The next rows could not be read."
        loadingLabel="Reading more rows…"
      >
        {(rows) => <p>{rows.join(", ")}</p>}
      </CardPagedList>,
    );
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("holds rows while the first page is read, and says when there is nothing", () => {
    const { container, rerender } = draw(list({ rows: undefined, pending: true }));
    expect(container.querySelector(".dk-rec-wait")).not.toBeNull();

    rerender(
      <CardPagedList
        list={list({ rows: [], total: 0, hasMore: false })}
        emptyText="Nothing here yet."
        errorText="The rows could not be read."
        nextErrorText="The next rows could not be read."
        loadingLabel="Reading more rows…"
      >
        {() => null}
      </CardPagedList>,
    );
    expect(screen.getByText("Nothing here yet.")).toBeInTheDocument();
  });

  it("says a first page that failed in the list's place, and a later one under the rows it has", () => {
    const { rerender } = draw(list({ rows: undefined, error: new Error("Gateway down") }));
    expect(screen.getByText("Gateway down")).toBeInTheDocument();
    expect(screen.queryByRole("listitem")).toBeNull();

    rerender(
      <CardPagedList
        list={list({ error: new Error("Page two failed") })}
        emptyText="Nothing here yet."
        errorText="The rows could not be read."
        nextErrorText="The next rows could not be read."
        loadingLabel="Reading more rows…"
      >
        {(rows) => <p>{rows.join(", ")}</p>}
      </CardPagedList>,
    );
    expect(screen.getByText("one, two")).toBeInTheDocument();
    expect(screen.getByText("Page two failed")).toBeInTheDocument();
  });
});

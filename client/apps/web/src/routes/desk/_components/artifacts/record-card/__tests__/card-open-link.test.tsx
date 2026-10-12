import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { CardOpenLink } from "../record-card";

describe("CardOpenLink", () => {
  it("opens the record in the card's place when its kind has a card", () => {
    const onOpen = vi.fn();
    render(
      <MemoryRouter>
        <CardOpenLink to="/dispatch/locations?id=loc_1" onOpen={onOpen}>
          Open location
        </CardOpenLink>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open location" }));
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("links to the record's page when its kind has no card here", () => {
    render(
      <MemoryRouter>
        <CardOpenLink to="/dispatch/locations?id=loc_1">Open location</CardOpenLink>
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: /Open location/ })).toHaveAttribute(
      "href",
      "/dispatch/locations?id=loc_1",
    );
  });
});

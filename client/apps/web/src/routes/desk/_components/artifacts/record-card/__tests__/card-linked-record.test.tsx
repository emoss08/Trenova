import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation } from "@trenova/shared/types/permission";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CardOpenLink, CardRecordRow } from "../record-card";

afterEach(() => {
  cleanup();
  clearPermissions();
});

describe("a linked record on a card", () => {
  it("opens a linked worker in the card's place for someone who may read workers", () => {
    seedPermissions({ worker: Operation.Read });
    const onOpen = vi.fn();
    render(
      <MemoryRouter>
        <CardRecordRow kind="worker" to="/hr/workers?panelEntityId=wrk_1" onOpen={onOpen}>
          <b>Rosa Diaz</b>
        </CardRecordRow>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Rosa Diaz" }));
    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("draws a linked worker unlinked, with a lock, for someone who may not read workers", () => {
    seedPermissions({ tractor: Operation.Read });
    const onOpen = vi.fn();
    const { container } = render(
      <MemoryRouter>
        <CardRecordRow kind="worker" to="/hr/workers?panelEntityId=wrk_1" onOpen={onOpen}>
          <b>Rosa Diaz</b>
        </CardRecordRow>
      </MemoryRouter>,
    );

    expect(screen.queryByRole("button", { name: "Rosa Diaz" })).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
    const row = container.querySelector('[data-slot="locked-reference"]');
    expect(row?.className).toContain("dk-rec-row");
    expect(row?.getAttribute("aria-description")).toBe("You don't have access to workers");
  });

  it("leaves out the way to a linked carrier for someone who may not read carriers", () => {
    seedPermissions({ carrier_settlement: Operation.Read });
    const { container } = render(
      <MemoryRouter>
        <CardOpenLink kind="carrier" to="/carriers?panelEntityId=car_1">
          Open carrier
        </CardOpenLink>
      </MemoryRouter>,
    );

    expect(container.textContent).toBe("");
  });

  it("keeps the way to a linked carrier for someone who may read carriers", () => {
    seedPermissions({ carrier: Operation.Read });
    render(
      <MemoryRouter>
        <CardOpenLink kind="carrier" to="/carriers?panelEntityId=car_1">
          Open carrier
        </CardOpenLink>
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: /Open carrier/ })).toHaveAttribute(
      "href",
      "/carriers?panelEntityId=car_1",
    );
  });
});

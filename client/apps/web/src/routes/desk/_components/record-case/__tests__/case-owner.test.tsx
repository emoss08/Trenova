import type { DeskCaseOwnership } from "@/lib/graphql/desk-case";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render as renderBare, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactElement, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CaseOwner } from "../case-owner";

const assignDeskCase = vi.fn(async () => true);
const releaseDeskCase = vi.fn(async () => true);

vi.mock("@/lib/graphql/desk-case", () => ({
  assignDeskCase: (...args: unknown[]) => assignDeskCase(...(args as [])),
  releaseDeskCase: (...args: unknown[]) => releaseDeskCase(...(args as [])),
}));
vi.mock("@/components/resolved-user-avatar", () => ({
  ResolvedUserAvatar: () => <i data-testid="avatar" />,
}));
vi.mock("@/components/autocomplete-fields", () => ({
  UserAutocompleteField: () => <input aria-label="Owner" />,
}));

const ME = "usr_me";
const DANA = { id: "usr_dana", name: "Dana Whitfield" };

function ownership(overrides: Partial<DeskCaseOwnership> = {}): DeskCaseOwnership {
  return {
    owner: null,
    auto: false,
    covering: false,
    canWork: true,
    canClaim: true,
    canAssign: false,
    ...overrides,
  };
}

function render(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );

  return renderBare(ui, { wrapper });
}

function draw(state: DeskCaseOwnership, onChanged = vi.fn(async () => undefined)) {
  render(
    <CaseOwner kind="shipment" id="shp_1" label="SHP-1" ownership={state} onChanged={onChanged} />,
  );

  return onChanged;
}

beforeEach(() => {
  useAuthStore.setState({ user: { id: ME } as never });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("whose case it is", () => {
  it("lets a person who works cases take an unowned one", async () => {
    const onChanged = draw(ownership());

    await userEvent.click(screen.getByRole("button", { name: "Case owner: Unowned" }));
    await userEvent.click(await screen.findByText("Take this case"));

    await waitFor(() => expect(assignDeskCase).toHaveBeenCalledWith("shipment", "shp_1", null));
    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
    expect(screen.queryByText("Assign to someone")).toBeNull();
  });

  it("names another person's case and offers nothing to someone who cannot assign", () => {
    draw(ownership({ owner: DANA, canWork: false, canClaim: false }));

    expect(screen.getByText("Owned by Dana Whitfield")).toBeTruthy();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("says when the reader is covering for the owner", () => {
    draw(ownership({ owner: DANA, covering: true, canClaim: false }));

    expect(screen.getByText("Covering for Dana Whitfield")).toBeTruthy();
  });

  it("lets the owner let their case go", async () => {
    draw(ownership({ owner: { id: ME, name: "Me" }, canClaim: false }));

    await userEvent.click(screen.getByRole("button", { name: "Case owner: Your case" }));
    await userEvent.click(await screen.findByText("Let it go"));

    await waitFor(() => expect(releaseDeskCase).toHaveBeenCalledWith("shipment", "shp_1"));
    expect(assignDeskCase).not.toHaveBeenCalled();
  });

  it("lets someone who assigns cases take over, hand on or free another person's case", async () => {
    draw(ownership({ owner: DANA, canWork: false, canClaim: false, canAssign: true }));

    await userEvent.click(
      screen.getByRole("button", { name: "Case owner: Owned by Dana Whitfield" }),
    );

    expect(await screen.findByText("Take it over")).toBeTruthy();
    expect(screen.getByText("Assign to someone")).toBeTruthy();
    expect(screen.getByText("Leave it unowned")).toBeTruthy();
  });
});

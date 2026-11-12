import type { DeskCaseOwnership } from "@/lib/graphql/desk-case";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useClaimOnOpen } from "../use-claim-on-open";

const mocks = vi.hoisted(() => ({ assign: vi.fn(), failed: vi.fn() }));

vi.mock("@/lib/graphql/desk-case", () => ({ assignDeskCase: mocks.assign }));
vi.mock("@/hooks/use-api-mutation", () => ({ handleMutationError: mocks.failed }));

afterEach(() => vi.clearAllMocks());

function ownership(overrides: Partial<DeskCaseOwnership> = {}): DeskCaseOwnership {
  return {
    owner: null,
    auto: false,
    covering: false,
    canWork: true,
    canClaim: true,
    canAssign: false,
    ...overrides,
  } as DeskCaseOwnership;
}

function open(props: { ownership: DeskCaseOwnership | undefined; enabled: boolean }) {
  const onClaimed = vi.fn();
  const hook = renderHook(
    (current: typeof props) =>
      useClaimOnOpen({ kind: "shipment", id: "shp_1", onClaimed, ...current }),
    { initialProps: props },
  );

  return { onClaimed, ...hook };
}

/** Opening a case from the Watchtower to work it makes it the opener's, once. */
describe("claiming a case as it is opened to be worked", () => {
  it("takes an unowned case for the person who opened it, and reads the case again", async () => {
    mocks.assign.mockResolvedValue({});
    const { onClaimed, rerender } = open({ ownership: undefined, enabled: true });
    expect(mocks.assign).not.toHaveBeenCalled();

    rerender({ ownership: ownership(), enabled: true });
    rerender({ ownership: ownership(), enabled: true });

    await waitFor(() => expect(onClaimed).toHaveBeenCalledOnce());
    expect(mocks.assign).toHaveBeenCalledExactlyOnceWith("shipment", "shp_1", null);
  });

  it("leaves someone else's case with them, even for a lead who could take it over", () => {
    open({
      ownership: ownership({
        owner: { id: "usr_2", name: "Dana Ruiz" } as DeskCaseOwnership["owner"],
        canClaim: false,
        canAssign: true,
      }),
      enabled: true,
    });

    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("takes nothing when the case was only opened to be read, or cannot be claimed", () => {
    open({ ownership: ownership(), enabled: false });
    open({ ownership: ownership({ canClaim: false }), enabled: true });

    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("says so when the claim is refused, and does not ask again", async () => {
    mocks.assign.mockRejectedValue(new Error("taken"));
    const { onClaimed, rerender } = open({ ownership: ownership(), enabled: true });
    rerender({ ownership: ownership(), enabled: true });

    await waitFor(() => expect(mocks.failed).toHaveBeenCalledOnce());
    expect(mocks.assign).toHaveBeenCalledOnce();
    expect(onClaimed).not.toHaveBeenCalled();
  });
});

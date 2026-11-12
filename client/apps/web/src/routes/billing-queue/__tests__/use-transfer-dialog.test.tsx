import { act, renderHook } from "@testing-library/react";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { useTransferDialog } from "../use-transfer-dialog";

function renderWithUrl(
  searchParams: string,
  onUrlUpdate = vi.fn<(event: UrlUpdateEvent) => void>(),
) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate} hasMemory>
      {children}
    </NuqsTestingAdapter>
  );
  return { ...renderHook(() => useTransferDialog(), { wrapper }), onUrlUpdate };
}

describe("useTransferDialog", () => {
  it("opens on the run a link names", () => {
    const { result } = renderWithUrl("?transferRun=btr_1");

    expect(result.current.open).toBe(true);
    expect(result.current.runId).toBe("btr_1");
  });

  it("lets go of the run when the dialog closes, so a refresh does not reopen it", async () => {
    const { result, onUrlUpdate } = renderWithUrl("?transferRun=btr_1");

    act(() => result.current.onOpenChange(false));

    await vi.waitFor(() => expect(result.current.runId).toBeNull());
    expect(result.current.open).toBe(false);
    await vi.waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    expect(onUrlUpdate.mock.lastCall?.[0].searchParams.has("transferRun")).toBe(false);
  });

  it("opens and remembers a run once one starts", async () => {
    const { result, onUrlUpdate } = renderWithUrl("");
    expect(result.current.open).toBe(false);

    act(() => result.current.onRunIdChange("btr_2"));

    await vi.waitFor(() => expect(result.current.runId).toBe("btr_2"));
    expect(result.current.open).toBe(true);
    await vi.waitFor(() => expect(onUrlUpdate).toHaveBeenCalled());
    expect(onUrlUpdate.mock.lastCall?.[0].searchParams.get("transferRun")).toBe("btr_2");
  });
});

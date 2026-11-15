import { createTestRouter, TestRouter } from "@trenova/shared/testing/test-router";
import { act, renderHook } from "@testing-library/react";
import type { HistoryState } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { useQueryParams } from "../use-query-params";

// What an app keeps on a history entry (the Desk marks a case opened to be worked).
const CARRIED = { startCase: true } as HistoryState;

function setup(initial: string) {
  const router = createTestRouter(undefined, [initial]);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestRouter router={router}>{children}</TestRouter>
  );
  const hook = renderHook(() => useQueryParams(), { wrapper });
  return { router, hook };
}

describe("useQueryParams", () => {
  it("reads the address's query", () => {
    const { hook } = setup("/inbox?folder=waiting&message=msg_1");

    expect(hook.result.current[0].get("folder")).toBe("waiting");
    expect(hook.result.current[0].get("message")).toBe("msg_1");
  });

  it("changes the query in place, keeping the path and the rest of the query", async () => {
    const { router, hook } = setup("/inbox?folder=waiting#reader");

    await act(async () => {
      hook.result.current[1]((current) => {
        const next = new URLSearchParams(current);
        next.set("message", "msg_2");
        return next;
      });
    });

    expect(router.state.location.pathname).toBe("/inbox");
    expect(router.state.location.searchStr).toBe("?folder=waiting&message=msg_2");
    expect(router.state.location.hash).toBe("reader");
    expect(hook.result.current[0].get("message")).toBe("msg_2");
  });

  it("starts each change from the address as it stands, so quick changes keep each other", async () => {
    const { router, hook } = setup("/inbox");
    const [, setQueryParams] = hook.result.current;

    await act(async () => {
      setQueryParams((current) => new URLSearchParams([...current, ["a", "1"]]));
      setQueryParams((current) => new URLSearchParams([...current, ["b", "2"]]));
    });

    expect(router.state.location.searchStr).toBe("?a=1&b=2");
  });

  it("replaces the query when handed a whole one", async () => {
    const { router, hook } = setup("/insights?severity=critical");

    await act(async () => {
      hook.result.current[1]({ category: "billing" });
    });

    expect(router.state.location.searchStr).toBe("?category=billing");
  });

  it("replaces the history entry when asked, and keeps what the entry carried", async () => {
    const { router, hook } = setup("/desk/needs");
    await act(() =>
      router.navigate({ href: "/desk/r/shipment/shp_1", state: CARRIED }),
    );
    const entry = router.state.location.state.__TSR_index;

    await act(async () => {
      hook.result.current[1]({ tab: "notes" }, { replace: true });
    });

    expect(router.state.location.searchStr).toBe("?tab=notes");
    expect(router.state.location.state.__TSR_index).toBe(entry);
    expect(router.state.location.state).toMatchObject({ startCase: true });
  });
});

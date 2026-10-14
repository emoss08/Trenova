import { analytics } from "@/lib/queries/analytics";
import { attention } from "@/lib/queries/attention";
import { homeLayout } from "@/lib/queries/home-layout";
import { prefetch } from "@/routes/home/page";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { LoaderFunctionArgs } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The manifest is GET /me/permissions over REST; the rate limiter's problem document
// carries Retry-After.
function answerManifestWith(response: Response) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(input instanceof Request ? input.url : String(input), "http://localhost");
      if (!url.pathname.endsWith("/me/permissions")) {
        throw new Error(`unexpected request to ${url.pathname}`);
      }
      return response;
    }),
  );
}

const loaderArgs = {
  request: new Request("http://localhost/"),
  params: {},
  context: undefined,
} as unknown as LoaderFunctionArgs;

describe("home prefetch without a manifest loaded", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    usePermissionStore.getState().clearPermissions();
  });

  // The prefetch only warms the cache. A manifest it cannot load costs the shipment
  // analytics it would have warmed, not the layout and the attention summary.
  it("still warms the layout and the attention summary when the manifest is rate-limited", async () => {
    answerManifestWith(
      new Response(
        JSON.stringify({
          type: "https://api.trenova.app/problems/rate-limit-exceeded",
          title: "Rate Limit Exceeded",
          status: 429,
          detail: "Too many requests",
        }),
        {
          status: 429,
          headers: { "Content-Type": "application/problem+json", "Retry-After": "7" },
        },
      ),
    );

    const list = await prefetch(loaderArgs);

    expect(list.map((entry) => entry.queryKey)).toEqual([
      homeLayout.effective().queryKey,
      attention.summary().queryKey,
    ]);
    expect(list.map((entry) => entry.queryKey)).not.toContainEqual(
      analytics.get("shipment-management").queryKey,
    );
  });
});

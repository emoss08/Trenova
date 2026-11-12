import { vi } from "vitest";
import type {
  BatchProposalPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";

/**
 * The test environment's IntersectionObserver never reports, so a sheet
 * would never see its rows on screen. This one reports every row it is given
 * as in view at once, as a sheet short enough to fit does.
 */
export function stubRowsInView() {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      callback: IntersectionObserverCallback;
      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback;
      }
      observe(target: Element) {
        this.callback(
          [{ isIntersecting: true, target } as unknown as IntersectionObserverEntry],
          this as unknown as IntersectionObserver,
        );
      }
      unobserve() {}
      disconnect() {}
    },
  );
}

/**
 * The sheet's batch read answered from a test's one-proposal preview mock,
 * the way the server previews each proposal of a batch on its own: a preview
 * that fails becomes that row's error and never the others'.
 */
export function batchFrom(
  one: (request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>,
) {
  return async (ids: readonly string[]): Promise<BatchProposalPreview[]> =>
    Promise.all(
      ids.map(async (id) => {
        try {
          const preview = await one({ scope: "approver", id, modifications: null });
          return { proposalId: id, preview, error: null };
        } catch {
          return { proposalId: id, preview: null, error: "This change could not be previewed." };
        }
      }),
    );
}

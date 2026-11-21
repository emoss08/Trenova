import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { domMax, LazyMotion } from "motion/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import DocumentIntelligenceForm from "../_components/document-intelligence-form";
import { DocumentIntelligenceSkeleton } from "../skeleton";
import { savedDocumentControl } from "./document-intelligence-fixture";

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    get: Object.assign(
      () => ({ queryKey: ["documentControl", "get"], queryFn: async () => savedDocumentControl }),
      { _def: ["documentControl", "get"] },
    ),
    extractionStatus: Object.assign(
      () => ({
        queryKey: ["documentControl", "extraction-status"],
        queryFn: async () => ({
          modelProviderAvailable: true,
          modelProviderReason: "available",
          converterAvailable: true,
          extractionAvailable: true,
        }),
      }),
      { _def: ["documentControl", "extraction-status"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <DocumentIntelligenceSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

async function formOutline(search = ""): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <DocumentIntelligenceForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
  await screen.findByText("Read uploaded documents");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("DocumentIntelligenceSkeleton", () => {
  it("draws every tab, and Reading's section with its rows closed", async () => {
    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 3, sections: 1, rows: 1 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it.each([
    ["filing", { tabs: 3, sections: 2, rows: 2 }],
    ["drafts", { tabs: 3, sections: 2, rows: 2 }],
  ])("draws the %s tab the address opens, as the form does", async (tab, expected) => {
    const outline = skeletonOutline(`?tab=${tab}`);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(`?tab=${tab}`)).toEqual(outline);
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <DocumentIntelligenceSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});

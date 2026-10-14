import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it } from "vitest";
import { AIControlTabSkeleton } from "../ai-control-tab-skeleton";
import { aiControlTabValues, type AIControlTab, type RailView } from "../ai-control-tabs";

afterEach(() => {
  cleanup();
});

const CASES: Record<AIControlTab, { view: RailView | null; label: string }> = {
  overview: { view: null, label: "Loading the overview" },
  agents: { view: null, label: "Loading agents" },
  providers: { view: null, label: "Loading providers" },
  extensions: { view: null, label: "Loading extensions" },
  memory: { view: null, label: "Loading memory" },
  retrieval: { view: null, label: "Loading retrieval" },
  safety: { view: "rules", label: "Loading safety" },
  quality: { view: "agents", label: "Loading quality" },
  analytics: { view: null, label: "Loading analytics" },
  activity: { view: "runs", label: "Loading activity" },
  audit: { view: "trail", label: "Loading the audit trail" },
};

describe("AIControlTabSkeleton", () => {
  it.each(aiControlTabValues)("opens the %s tab on its own layout, not a bare table", (tab) => {
    const { view, label } = CASES[tab];
    const { container } = render(
      <NuqsTestingAdapter>
        <AIControlTabSkeleton tab={tab} view={view} />
      </NuqsTestingAdapter>,
    );

    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", label);
    expect(screen.getAllByLabelText(label)[0]).toBe(root);
    expect(root?.getAttribute("data-slot")).not.toBe("data-table");
    expect(container.textContent).toBe("");
  });
});

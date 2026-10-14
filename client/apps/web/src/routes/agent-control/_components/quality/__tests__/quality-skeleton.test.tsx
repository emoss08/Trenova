import { cleanup, render } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import type { QualityView } from "../../../ai-control-tabs";
import { QualityTabSkeleton } from "../quality-skeleton";

afterEach(() => {
  cleanup();
});

function renderSkeleton(view: QualityView, searchParams = "") {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <NuqsTestingAdapter searchParams={searchParams}>{children}</NuqsTestingAdapter>
  );
  return render(<QualityTabSkeleton view={view} />, { wrapper });
}

describe("QualityTabSkeleton", () => {
  it("heads the agents with Nova's sentence and five figures over a four-column table", () => {
    const { container } = renderSkeleton("agents");

    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", "Loading quality");
    expect(root?.querySelectorAll(".hero")).toHaveLength(1);
    expect(root?.querySelectorAll(".figs .fg")).toHaveLength(5);
    expect(root?.querySelectorAll("th")).toHaveLength(4);
  });

  it("draws the golden set as a five-column table without the head", () => {
    const { container } = renderSkeleton("golden");

    expect(container.querySelector(".hero")).toBeNull();
    expect(container.querySelector(".figs")).toBeNull();
    expect(container.querySelectorAll("th")).toHaveLength(5);
  });

  it("draws a suite run's cases in place of the runs when the address names one", () => {
    expect(renderSkeleton("runs").container.querySelectorAll("th")).toHaveLength(5);
    cleanup();
    expect(renderSkeleton("runs", "?suiteRun=run_1").container.querySelectorAll("th")).toHaveLength(
      4,
    );
  });

  it("draws document extraction's figures and accuracy panels, not a table", () => {
    const { container } = renderSkeleton("extraction");

    expect(container.querySelector(".hero")).toBeNull();
    expect(container.querySelectorAll('[data-slot="kpi-strip"]')).toHaveLength(1);
    expect(container.querySelector("table")).toBeNull();
  });

  it("draws the extraction view the address names", () => {
    const { container } = renderSkeleton("extraction", "?extraction=corrections");

    expect(container.querySelectorAll("th")).toHaveLength(7);
  });

  it("puts no visible text on the page", () => {
    expect(renderSkeleton("agents").container.textContent).toBe("");
  });
});

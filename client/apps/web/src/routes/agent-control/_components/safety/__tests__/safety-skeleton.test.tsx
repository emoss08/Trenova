import { cleanup, render } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import type { SafetyView } from "../../../ai-control-tabs";
import { SafetyTabSkeleton } from "../safety-skeleton";

afterEach(() => {
  cleanup();
});

function renderSkeleton(view: SafetyView, searchParams = "") {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <NuqsTestingAdapter searchParams={searchParams}>{children}</NuqsTestingAdapter>
  );
  return render(<SafetyTabSkeleton view={view} />, { wrapper });
}

describe("SafetyTabSkeleton", () => {
  it("draws the head, the two-way view switch and the five-column tool rules", () => {
    const { container } = renderSkeleton("rules");

    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", "Loading safety");
    expect(root?.querySelectorAll(".figs .fg")).toHaveLength(3);
    expect(root?.querySelectorAll(".egm .egm-s").length).toBeGreaterThan(0);
    expect(root?.querySelectorAll(".vw .seg > *")).toHaveLength(2);
    expect(root?.querySelectorAll("th")).toHaveLength(5);
  });

  it("offers the roster to pick from when no agent is picked", () => {
    const { container } = renderSkeleton("agents");

    expect(container.querySelector(".pk-e .ms")).not.toBeNull();
    expect(container.querySelector("table")).toBeNull();
  });

  it("draws a header for each picked agent over their six-column table", () => {
    const { container } = renderSkeleton("agents", "?safetyAgents=agent_1,agent_2");

    expect(container.querySelector(".pk-e")).toBeNull();
    expect(container.querySelectorAll(".ahs .ah")).toHaveLength(2);
    expect(container.querySelectorAll("th")).toHaveLength(6);
  });

  it("draws no more agents than can be compared", () => {
    const { container } = renderSkeleton("agents", "?safetyAgents=a,b,c,d,e,a");

    expect(container.querySelectorAll(".ahs .ah")).toHaveLength(4);
  });

  it("puts no visible text on the page", () => {
    expect(renderSkeleton("rules").container.textContent).toBe("");
  });
});

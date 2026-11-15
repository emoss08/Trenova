import { BASE_ONLY_UNAVAILABLE, seedPlan } from "@/test/plan";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FleetSafetyPage } from "../page";

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({ children }: { children: ReactNode }) => <main>{children}</main>,
}));
vi.mock("../_components/fleet-safety-console", () => ({
  default: () => <p>fleet console</p>,
}));
vi.mock("../_components/my-dot-intelligence", () => ({
  MyDotIntelligence: () => <p>my dot</p>,
}));

function renderPage(searchParams: string) {
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <FleetSafetyPage />
    </NuqsTestingAdapter>,
  );
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("fleet safety's My DOT view", () => {
  it("opens a link to My DOT on the fleet when carrier intelligence is not on the plan", async () => {
    seedPlan({ [Resource.WorkerSafetyEvent]: Operation.Read }, BASE_ONLY_UNAVAILABLE);

    renderPage("?view=my-dot");

    expect(await screen.findByText("fleet console")).toBeInTheDocument();
    expect(screen.queryByText("my dot")).toBeNull();
    expect(screen.queryByRole("tab", { name: "My DOT" })).toBeNull();
  });

  it("opens My DOT for a reader who holds carrier intelligence", async () => {
    seedPlan(
      {
        [Resource.WorkerSafetyEvent]: Operation.Read,
        [Resource.CarrierIntelligence]: Operation.Read,
      },
      [],
    );

    renderPage("?view=my-dot");

    expect(await screen.findByText("my dot")).toBeInTheDocument();
  });
});

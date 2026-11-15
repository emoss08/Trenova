import { tableConfiguration } from "@/lib/queries/table-configuration";
import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { cleanup, render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import type { LoaderFunctionArgs } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { prefetch, WorkersPage } from "../page";

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({ children }: { children: ReactNode }) => <main>{children}</main>,
}));
vi.mock("../_components/pto-content", () => ({
  PTOContent: () => <div data-testid="pto-overview" />,
}));
vi.mock("../_components/worker-table", () => ({
  default: () => <div data-testid="worker-table" />,
}));
vi.mock("../_components/pto/pto-table", () => ({
  default: () => <div data-testid="pto-table" />,
}));
vi.mock("../_components/pto/pto-balance-summary-card", () => ({
  PTOBalanceSummaryCard: () => <div data-testid="pto-balance-summary" />,
}));

const GRANTS = {
  [Resource.Worker]: Operation.Read,
  [Resource.WorkerPTO]: Operation.Read | Operation.Manage,
};

function renderPage(searchParams = "") {
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <WorkersPage />
    </NuqsTestingAdapter>,
  );
}

async function prefetched(search: string) {
  return await prefetch({
    request: new Request(`http://localhost/hr/workers${search}`),
    params: {},
    context: undefined,
  } as unknown as LoaderFunctionArgs);
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("the workers page on a plan without time off", () => {
  it("draws the roster alone, with no time-off charts and no time-off tab", async () => {
    seedPlan(GRANTS, BASE_ONLY_UNAVAILABLE);

    renderPage("?pageTab=pto");

    expect(await screen.findByTestId("worker-table")).toBeTruthy();
    expect(screen.queryByTestId("pto-overview")).toBeNull();
    expect(screen.queryByRole("tab")).toBeNull();
    expect(screen.queryByTestId("pto-balance-summary")).toBeNull();
    expect(screen.queryByTestId("pto-table")).toBeNull();
  });

  it("warms only the roster's table configuration, never a time-off read", async () => {
    seedPlan(GRANTS, unavailableFeatures("workforce.time_off"));

    const list = await prefetched("?pageTab=pto");

    expect(list.map((entry) => entry.queryKey)).toEqual([
      tableConfiguration.default("Worker").queryKey,
    ]);
  });
});

describe("the workers page on a plan with time off", () => {
  it("draws the time-off overview and both tabs", async () => {
    seedPlan(GRANTS, unavailableFeatures("ai.desk"));

    renderPage();

    expect(screen.getByTestId("pto-overview")).toBeTruthy();
    expect(screen.getByRole("tab", { name: "Workers" })).toBeTruthy();
    expect(screen.getByRole("tab", { name: "Paid time off" })).toBeTruthy();
    expect(await screen.findByTestId("worker-table")).toBeTruthy();
  });

  it("warms the time-off chart beside the roster", async () => {
    seedPlan(GRANTS, []);

    const list = await prefetched("");

    expect(list.length).toBeGreaterThan(1);
    expect(list.map((entry) => entry.queryKey)).toContainEqual(
      tableConfiguration.default("Worker").queryKey,
    );
  });
});

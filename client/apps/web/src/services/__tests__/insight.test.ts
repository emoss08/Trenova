import { beforeEach, describe, expect, it, vi } from "vitest";
import { activeInsightsInput, InsightService } from "../insight";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

const insight = {
  id: "inst_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  detectorKey: "unbilled_detention",
  category: "CashFlow",
  severity: "Warning",
  status: "Active",
  dedupeKey: "unbilled_detention:loc_1",
  subject: "Dock 4",
  headline: "Detention at Dock 4 is unbilled",
  narrative: "",
  recommendation: "",
  narrated: false,
  metrics: [
    {
      key: "amount",
      label: "Unbilled",
      value: "1250.5",
      unit: "Currency",
      direction: "HigherIsWorse",
      baseline: null,
      baselineLabel: "",
    },
  ],
  links: null,
  windowStart: 100,
  windowEnd: 200,
  detectedAt: 200,
  staleAt: 300,
  dismissedAt: null,
  dismissedById: "",
  dismissReason: "",
  modelIdentifier: "",
  providerId: "",
  version: 1,
  createdAt: 200,
  updatedAt: 200,
};

beforeEach(() => {
  requestGraphQL.mockReset();
});

describe("activeInsightsInput", () => {
  it("names the surface a page is asking for", () => {
    expect(activeInsightsInput(4, "Accounting")).toEqual({ limit: 4, surface: "Accounting" });
  });

  it("asks for the whole view when no surface is given", () => {
    expect(activeInsightsInput(6)).toEqual({ limit: 6 });
  });
});

describe("InsightService", () => {
  it("reads the active slice and keeps the list shape the widget reads", async () => {
    requestGraphQL.mockResolvedValue({ activeInsights: [insight] });
    const signal = new AbortController().signal;

    const result = await new InsightService().list(3, "Dispatch", { signal });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ActiveInsights",
        variables: { input: { limit: 3, surface: "Dispatch" } },
        signal,
      }),
    );
    expect(result.results).toHaveLength(1);
    expect(result.results[0]?.metrics?.[0]?.value).toBe("1250.5");
  });

  it("browses only the lifecycles this build knows", async () => {
    requestGraphQL.mockResolvedValue({
      insights: { results: [insight], total: 1, totalExact: true },
    });

    const page = await new InsightService().browse({
      statuses: ["Dismissed", "Archived"],
      categories: ["CashFlow"],
      severities: [],
      limit: 25,
      offset: 50,
    });

    expect(requestGraphQL.mock.calls[0]?.[0]).toMatchObject({
      operationName: "BrowseInsights",
      variables: {
        input: {
          statuses: ["Dismissed"],
          categories: ["CashFlow"],
          severities: [],
          limit: 25,
          offset: 50,
        },
      },
    });
    expect(page).toMatchObject({ total: 1, totalExact: true });
  });

  it("dismisses with the reason given", async () => {
    requestGraphQL.mockResolvedValue({
      dismissInsight: { ...insight, status: "Dismissed", dismissReason: "Known" },
    });

    const dismissed = await new InsightService().dismiss("inst_1", "Known");

    expect(requestGraphQL.mock.calls[0]?.[0]).toMatchObject({
      operationName: "DismissInsight",
      variables: { input: { insightId: "inst_1", reason: "Known" } },
    });
    expect(dismissed.status).toBe("Dismissed");
  });
});

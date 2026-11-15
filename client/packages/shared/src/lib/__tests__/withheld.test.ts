import { api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { requestGraphQL, requestGraphQLResult } from "@trenova/shared/lib/graphql";
import { NO_WITHHOLDINGS, planWithholdingAt, withholdingsOf } from "@trenova/shared/lib/withheld";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

function jsonResponse(body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json", ...headers },
  });
}

beforeEach(() => {
  clearCsrfToken();
  setCsrfToken("csrf");
});

afterEach(() => {
  vi.unstubAllGlobals();
  clearCsrfToken();
});

// docs/engineering/reference-views.md: a GraphQL response names what the plan withheld in
// extensions.withheldByPlan as { path, feature }, sorted, and what access withheld in
// extensions.withheldByAccess; paths carry no list indices.
describe("a GraphQL response's withholdings", () => {
  const response = {
    data: {
      invoice: {
        id: "inv_1",
        factoringCompany: null,
        customer: { id: "cus_1", creditLimit: null },
      },
    },
    extensions: {
      withheldByAccess: ["invoice.customer.creditLimit"],
      withheldByPlan: [
        { path: "invoice.factoringCompany", feature: "tms.factoring" },
        { path: "invoice.lines.detention", feature: "tms.detention" },
      ],
    },
  };

  it("comes back beside the data from requestGraphQLResult", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(response)),
    );

    const result = await requestGraphQLResult({ document: "query Invoice { invoice { id } }" });

    expect(result.withheld).toEqual({
      byAccess: ["invoice.customer.creditLimit"],
      byPlan: [
        { path: "invoice.factoringCompany", feature: "tms.factoring" },
        { path: "invoice.lines.detention", feature: "tms.detention" },
      ],
    });
  });

  it("is asked of the data requestGraphQL resolved to", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(response)),
    );

    const data = await requestGraphQL({ document: "query Invoice { invoice { id } }" });

    expect(planWithholdingAt(withholdingsOf(data), "invoice.factoringCompany")).toEqual({
      path: "invoice.factoringCompany",
      feature: "tms.factoring",
    });
    expect(planWithholdingAt(withholdingsOf(data), "invoice.customer.creditLimit")).toBeUndefined();
  });

  it("is empty for a response that withheld nothing, and for a copy of the data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ data: { invoice: { id: "inv_1" } } })),
    );
    const plain = await requestGraphQLResult({ document: "query Invoice { invoice { id } }" });
    expect(plain.withheld).toEqual(NO_WITHHOLDINGS);

    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(response)),
    );
    const data = await requestGraphQL<Record<string, unknown>>({
      document: "query Invoice { invoice { id } }",
    });
    expect(withholdingsOf({ ...data })).toEqual(NO_WITHHOLDINGS);
  });
});

// A REST response names them in headers: Withheld-By-Plan as path=feature, comma separated.
describe("a REST response's withholdings", () => {
  it("is asked of the body the request resolved to", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          { id: "inv_1", factoringCompany: { id: "fac_1" }, shipment: { moves: [] } },
          {
            "Withheld-By-Plan": "factoringCompany=tms.factoring, shipment.moves=tms.detention",
            "Withheld-By-Access": "shipment.customer",
          },
        ),
      ),
    );

    const body = await api.get<Record<string, unknown>>("/invoices/inv_1/");

    expect(withholdingsOf(body)).toEqual({
      byAccess: ["shipment.customer"],
      byPlan: [
        { path: "factoringCompany", feature: "tms.factoring" },
        { path: "shipment.moves", feature: "tms.detention" },
      ],
    });
  });

  it("is empty when the response names none", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ id: "inv_1" })),
    );

    expect(withholdingsOf(await api.get("/invoices/inv_1/"))).toEqual(NO_WITHHOLDINGS);
  });
});

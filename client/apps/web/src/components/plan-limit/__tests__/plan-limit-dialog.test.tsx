import { TestRouter } from "@trenova/shared/testing/test-router";
import { handlePlanLimit } from "@/lib/plan-limit-handler";
import { PLAN_LIMIT_QUIET_MS, decidePlanLimit, usePlanLimitStore } from "@/stores/plan-limit-store";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError, api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { isExplainedError } from "@trenova/shared/lib/explained-errors";
import {
  planLimitFromError,
  setPlanLimitHandler,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, type PermissionManifest } from "@trenova/shared/types/permission";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GENERIC_PLAN_LIMIT_WORDING, planLimitCopy } from "@/lib/plan-limit-copy";
import { PlanLimitDialogHost } from "../plan-limit-dialog";

const t = (message: string | null | undefined, ...args: unknown[]) =>
  (message ?? "").replace(/\{(\d+)\}/g, (_, index: string) => {
    const value = args[Number(index)];
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
  });

const shipmentsQuota: PlanLimitNotice = {
  kind: "quota",
  meter: "shipments.total",
  limit: 12,
  used: 12,
  plan: "free_demo",
  message: "",
};

const readOnly: PlanLimitNotice = {
  kind: "restricted",
  feature: "",
  addOn: "",
  reason: "subscription_read_only",
  plan: "free_demo",
  message: "",
};

describe("planLimitCopy", () => {
  it("names the meter and its figures for a lifetime quota, and how to free room", () => {
    const copy = planLimitCopy({ ...shipmentsQuota, plan: "team" }, t);
    expect(copy.title).toBe("Shipments limit reached");
    expect(copy.description).toBe("Shipments: 12 of 12 used (Team).");
    expect(copy.guidance).toBe("Delete records you no longer need to free room.");
    expect(copy.usage).toEqual({ used: 12, limit: 12, usedLabel: "12", limitLabel: "12" });
  });

  it("promises nothing about plans the install may not sell", () => {
    for (const notice of [shipmentsQuota, readOnly]) {
      const copy = planLimitCopy({ ...notice, plan: "team" }, t);
      expect(`${copy.title} ${copy.description} ${copy.guidance}`).not.toMatch(
        /demo|trial|coming soon/i,
      );
    }
  });

  it("formats storage in bytes and Document Intelligence pages as a monthly count", () => {
    expect(
      planLimitCopy(
        { ...shipmentsQuota, meter: "documents.storage_bytes", limit: 104857600, used: 104857600 },
        t,
      ).usage?.limitLabel,
    ).toBe("100 MB");
    const pages = planLimitCopy(
      { ...shipmentsQuota, meter: "document_intelligence.pages", limit: 500, used: 500 },
      t,
    );
    expect(pages.usage?.limitLabel).toBe("500");
    expect(pages.guidance).toBe("This limit resets at the start of next month.");
  });

  it("explains a per-file limit without a usage bar", () => {
    const copy = planLimitCopy(
      {
        ...shipmentsQuota,
        meter: "documents.file_bytes",
        limit: 10485760,
        used: 25000000,
        plan: "",
      },
      t,
    );
    expect(copy.description).toBe("Current plan allows files up to 10.0 MB. This one is larger.");
    expect(copy.guidance).toBe("Upload a smaller file, or split it into parts.");
    expect(copy.usage).toBeNull();
  });

  it("still explains a quota whose figures did not arrive", () => {
    const copy = planLimitCopy({ ...shipmentsQuota, limit: null, used: null, plan: "team" }, t);
    expect(copy.usage).toBeNull();
    expect(copy.description).toBe("Your organization has reached its Shipments limit (Team).");
  });

  it("explains that nothing can be saved rather than blaming the action", () => {
    const copy = planLimitCopy(readOnly, t);
    expect(copy.title).toBe("This workspace is read-only");
    expect(copy.description).toContain("nothing can be created or changed");
  });

  describe("for an add-on the plan leaves out", () => {
    afterEach(() => usePermissionStore.getState().clearPermissions());

    function holdManifest(canManage: boolean) {
      usePermissionStore.setState({
        manifest: {
          permissions: canManage ? { organization: Operation.Read | Operation.Update } : {},
          routeAccess: {},
          unavailableFeatures: [
            {
              feature: "tms.factoring",
              featureName: "Factoring",
              addOn: "ACC",
              addOnName: "Accounting & Cash",
            },
          ],
        } as unknown as PermissionManifest,
      });
    }

    const factoring: PlanLimitNotice = {
      kind: "restricted",
      feature: "tms.factoring",
      addOn: "Accounting & Cash",
      reason: "plan",
      plan: "",
      message: "",
    };

    it("names the feature and its add-on, and sends the person to an administrator", () => {
      holdManifest(false);
      const copy = planLimitCopy(factoring, t);
      expect(copy.title).toBe("Factoring is not on your plan");
      expect(copy.description).toBe(
        "Factoring is part of the Accounting & Cash add-on, which is not on your plan.",
      );
      expect(copy.guidance).toBe("Ask your organization's administrator about adding it.");
      expect(copy.usage).toBeNull();
    });

    it("tells someone who manages the organization to add it themselves", () => {
      holdManifest(true);
      expect(planLimitCopy(factoring, t).guidance).toBe("Add it from your Trenova account.");
    });

    it("never shows the feature key or the add-on code", () => {
      holdManifest(false);
      const copy = planLimitCopy(factoring, t);
      const text = `${copy.title} ${copy.description} ${copy.guidance}`;
      expect(text).not.toContain("tms.factoring");
      expect(text).not.toContain("ACC");
    });

    it("still names the add-on when the manifest does not list the feature yet", () => {
      const copy = planLimitCopy(factoring, t);
      expect(copy.title).toBe("Not on your plan");
      expect(copy.description).toBe(
        "This is part of the Accounting & Cash add-on, which is not on your plan.",
      );
    });
  });

  it("falls back to a readable label for a meter the client does not know", () => {
    expect(planLimitCopy({ ...shipmentsQuota, meter: "edi.partners" }, t).title).toBe(
      "Edi partners limit reached",
    );
  });

  it("takes an edition's wording in place of the host's", () => {
    const copy = planLimitCopy(shipmentsQuota, t, {
      ...GENERIC_PLAN_LIMIT_WORDING,
      planName: () => "Starter",
      quotaGuidance: () => "Upgrade.",
    });
    expect(copy.description).toBe("Shipments: 12 of 12 used (Starter).");
    expect(copy.guidance).toBe("Upgrade.");
  });
});

describe("decidePlanLimit", () => {
  const base = {
    notice: shipmentsQuota,
    request: "write" as const,
    isAuthenticated: true,
    openNotice: null,
    lastDismissed: null,
    now: 100_000,
  };

  it("opens for a signed-in person", () => {
    expect(decidePlanLimit(base)).toBe("open");
  });

  it("leaves refusals on public pages to the page", () => {
    expect(decidePlanLimit({ ...base, isAuthenticated: false })).toBe("ignore");
  });

  it("leaves the signup wait list to the verification page", () => {
    expect(
      decidePlanLimit({
        ...base,
        notice: { ...readOnly, reason: "signups_paused" },
      }),
    ).toBe("ignore");
  });

  it("leaves an add-on's refusal of a read to the page that asked, and explains a write", () => {
    const addOn: PlanLimitNotice = {
      kind: "restricted",
      feature: "tms.factoring",
      addOn: "Accounting & Cash",
      reason: "plan",
      plan: "",
      message: "",
    };
    expect(decidePlanLimit({ ...base, notice: addOn, request: "read" })).toBe("ignore");
    expect(decidePlanLimit({ ...base, notice: addOn, request: "write" })).toBe("open");
    // A quota or a read-only workspace is the dialog's either way.
    expect(decidePlanLimit({ ...base, request: "read" })).toBe("open");
  });

  it("absorbs a second refusal while the dialog is open", () => {
    expect(decidePlanLimit({ ...base, openNotice: readOnly })).toBe("absorb");
  });

  it("stays quiet briefly after the same limit was dismissed, then opens again", () => {
    const lastDismissed = { key: "quota:shipments.total", at: base.now - 1_000 };
    expect(decidePlanLimit({ ...base, lastDismissed })).toBe("absorb");
    expect(
      decidePlanLimit({ ...base, lastDismissed, now: lastDismissed.at + PLAN_LIMIT_QUIET_MS }),
    ).toBe("open");
    expect(
      decidePlanLimit({
        ...base,
        lastDismissed: { key: "quota:customers.total", at: base.now - 1_000 },
      }),
    ).toBe("open");
  });
});

describe("PlanLimitDialogHost", () => {
  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("csrf");
    setPlanLimitHandler(handlePlanLimit);
    usePlanLimitStore.setState({ notice: null, lastDismissed: null });
    useAuthStore.setState({ isAuthenticated: true });
    usePermissionStore.setState({ hasPermission: () => true } as never);
  });

  afterEach(() => {
    setPlanLimitHandler(null);
    clearCsrfToken();
    vi.unstubAllGlobals();
    useAuthStore.setState({ isAuthenticated: false, user: null });
  });

  it("opens from a failed API call and suppresses the caller's toast for it", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              type: "https://trenova.app/problems/quota-exceeded",
              title: "Quota Exceeded",
              status: 402,
              code: "QUOTA_EXCEEDED",
              params: { meter: "customers.total", limit: "8", used: "8", plan: "free_demo" },
            }),
            { status: 402, headers: { "Content-Type": "application/json" } },
          ),
      ),
    );
    render(
      <TestRouter>
        <PlanLimitDialogHost plan={{ usagePath: "/admin/plan" }} />
      </TestRouter>,
    );

    let error: unknown;
    await act(async () => {
      error = await api.post("/customers/", { name: "Acme" }).catch((caught) => caught);
    });

    expect(error).toBeInstanceOf(ApiRequestError);
    expect(isExplainedError(error)).toBe(true);
    expect(await screen.findByText("Customers limit reached")).toBeInTheDocument();
    expect(screen.getByText("Customers: 8 of 8 used (Free demo).")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "View plan & usage" })).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Got it" }));
    expect(usePlanLimitStore.getState().notice).toBeNull();
    expect(usePlanLimitStore.getState().lastDismissed?.key).toBe("quota:customers.total");
  });

  it("offers no plan page without an edition that has one, and uses the edition's words", async () => {
    usePlanLimitStore.getState().show(shipmentsQuota);
    render(
      <TestRouter>
        <PlanLimitDialogHost
          plan={{
            limitCopy: (notice) =>
              notice.kind === "quota"
                ? {
                    title: "Edition title",
                    description: "Edition words",
                    guidance: "",
                    usage: null,
                  }
                : null,
          }}
        />
      </TestRouter>,
    );

    expect(await screen.findByText("Edition title")).toBeInTheDocument();
    expect(screen.getByText("Edition words")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "View plan & usage" })).not.toBeInTheDocument();
  });

  function answerGraphQLWithAddOnRefusal() {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              data: null,
              errors: [
                {
                  message:
                    "This feature is part of the Accounting & Cash add-on, which is not on your plan",
                  path: ["factoringCompanies"],
                  extensions: {
                    code: "PLAN_RESTRICTED",
                    type: "plan-restricted",
                    params: { feature: "tms.factoring", addOn: "Accounting & Cash" },
                  },
                },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          ),
      ),
    );
  }

  it("opens with the add-on's name for a GraphQL mutation an add-on refuses", async () => {
    answerGraphQLWithAddOnRefusal();
    render(
      <TestRouter>
        <PlanLimitDialogHost plan={{}} />
      </TestRouter>,
    );

    let error: unknown;
    await act(async () => {
      error = await requestGraphQL({
        document: "mutation AssignFactoring { assignFactoring { id } }",
      }).catch((caught) => caught);
    });

    expect(isExplainedError(error)).toBe(true);
    expect(
      await screen.findByText(
        "This is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
  });

  it("leaves a GraphQL query an add-on refuses to the page, without the dialog", async () => {
    answerGraphQLWithAddOnRefusal();
    render(
      <TestRouter>
        <PlanLimitDialogHost plan={{}} />
      </TestRouter>,
    );

    let error: unknown;
    await act(async () => {
      error = await requestGraphQL({
        document: "query FactoringCompanies { factoringCompanies { id } }",
      }).catch((caught) => caught);
    });

    expect(isExplainedError(error)).toBe(false);
    expect(usePlanLimitStore.getState().notice).toBeNull();
    expect(screen.queryByTestId("plan-limit-dialog")).toBeNull();
  });

  it("leaves a REST read an add-on refuses to the page, and explains a REST write", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              type: "https://trenova.app/problems/plan-restricted",
              title: "Plan Restricted",
              status: 403,
              code: "PLAN_RESTRICTED",
              params: { feature: "tms.factoring", addOn: "Accounting & Cash" },
            }),
            { status: 403, headers: { "Content-Type": "application/json" } },
          ),
      ),
    );
    render(
      <TestRouter>
        <PlanLimitDialogHost plan={{}} />
      </TestRouter>,
    );

    let read: unknown;
    await act(async () => {
      read = await api.get("/factoring-companies/").catch((caught) => caught);
    });
    expect(isExplainedError(read)).toBe(false);
    expect(usePlanLimitStore.getState().notice).toBeNull();

    let write: unknown;
    await act(async () => {
      write = await api.post("/factoring-companies/", { name: "Acme" }).catch((caught) => caught);
    });
    expect(isExplainedError(write)).toBe(true);
    expect(
      await screen.findByText(
        "This is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
  });

  it("does not open for a signed-out visitor, and leaves the error to its caller", async () => {
    useAuthStore.setState({ isAuthenticated: false });
    const error = new ApiRequestError(403, {
      type: "plan-restricted",
      title: "Plan Restricted",
      status: 403,
      code: "PLAN_RESTRICTED",
      params: { reason: "signups_paused" },
    });

    const notice = planLimitFromError(error);
    expect(notice).toMatchObject({ kind: "restricted", reason: "signups_paused" });
    expect(handlePlanLimit(notice as PlanLimitNotice, "write")).toBe(false);
    expect(usePlanLimitStore.getState().notice).toBeNull();
  });
});

import { handlePlanLimit } from "@/lib/plan-limit-handler";
import { PLAN_LIMIT_QUIET_MS, decidePlanLimit, usePlanLimitStore } from "@/stores/plan-limit-store";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError, api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import {
  isExplainedPlanLimitError,
  planLimitFromError,
  setPlanLimitHandler,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { MemoryRouter } from "react-router";
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
  capability: "",
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

  it("formats storage in bytes and spend in dollars", () => {
    expect(
      planLimitCopy(
        { ...shipmentsQuota, meter: "documents.storage_bytes", limit: 104857600, used: 104857600 },
        t,
      ).usage?.limitLabel,
    ).toBe("100 MB");
    const spend = planLimitCopy(
      { ...shipmentsQuota, meter: "ai.spend_cents", limit: 150, used: 150 },
      t,
    );
    expect(spend.usage?.limitLabel).toBe("$1.50");
    expect(spend.guidance).toBe("This limit resets at the start of next month.");
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

  it("names a restricted capability", () => {
    const copy = planLimitCopy(
      { ...readOnly, capability: "integrations", reason: "capability_restricted" },
      t,
    );
    expect(copy.title).toBe("Integrations not available");
    expect(copy.description).toBe("This part of Trenova is not included in your current plan.");
  });

  it("falls back to a readable label for a meter or capability the client does not know", () => {
    expect(planLimitCopy({ ...shipmentsQuota, meter: "edi.partners" }, t).title).toBe(
      "Edi partners limit reached",
    );
    expect(
      planLimitCopy({ ...readOnly, capability: "edi.partners", reason: "capability_restricted" }, t)
        .title,
    ).toBe("Edi partners not available");
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
      <MemoryRouter>
        <PlanLimitDialogHost plan={{ usagePath: "/admin/plan" }} />
      </MemoryRouter>,
    );

    let error: unknown;
    await act(async () => {
      error = await api.post("/customers/", { name: "Acme" }).catch((caught) => caught);
    });

    expect(error).toBeInstanceOf(ApiRequestError);
    expect(isExplainedPlanLimitError(error)).toBe(true);
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
      <MemoryRouter>
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
      </MemoryRouter>,
    );

    expect(await screen.findByText("Edition title")).toBeInTheDocument();
    expect(screen.getByText("Edition words")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "View plan & usage" })).not.toBeInTheDocument();
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
    expect(handlePlanLimit(notice as PlanLimitNotice)).toBe(false);
    expect(usePlanLimitStore.getState().notice).toBeNull();
  });
});

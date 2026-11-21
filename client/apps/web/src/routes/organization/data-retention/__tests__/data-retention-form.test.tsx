import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import DataRetentionForm from "../_components/data-retention-form";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...actual, api: { ...actual.api, get: mocks.get, put: mocks.put } };
});

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

/*
The GET /data-retention/ body, as dataretention.DataRetention serializes it
(services/tms/internal/modules/identity/domain/dataretention/dataretention.go). A
period of 0 on the EDI and evaluation-case fields keeps everything. */
const saved = {
  id: "dr_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  auditAccessRetentionDays: 90,
  auditOperationalRetentionDays: 730,
  auditFinancialRetentionDays: 2555,
  auditSecurityRetentionDays: 730,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 45,
  driverQualificationRetentionPeriod: 1095,
  agentEvalCaseRetentionPeriod: 365,
  aiFeedbackRetentionPeriod: 730,
  aiAuditRetentionPeriod: 2555,
  aiCorrectionRetentionPeriod: 730,
  version: 3,
  createdAt: 1_780_000_000,
  updatedAt: 1_780_000_000,
};

/** Noon UTC on 26 October 2026, so no time zone moves a cutoff to another day. */
const NOW = 1_793_016_000;

/** The body PUT /data-retention/ takes (dataretentionservice.UpdateDataRetentionRequest). */
const savedRequest = {
  auditAccessRetentionDays: 90,
  auditOperationalRetentionDays: 730,
  auditFinancialRetentionDays: 2555,
  auditSecurityRetentionDays: 730,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 45,
  driverQualificationRetentionPeriod: 1095,
  agentEvalCaseRetentionPeriod: 365,
  aiFeedbackRetentionPeriod: 730,
  aiAuditRetentionPeriod: 2555,
  aiCorrectionRetentionPeriod: 730,
};

function input(name: string) {
  return document.querySelector<HTMLInputElement>(`input[name="${name}"]`);
}

async function field(name: string) {
  return waitFor(() => {
    const found = input(name);
    expect(found).not.toBeNull();
    return found!;
  });
}

function row(label: string) {
  return within(screen.getByText(label).closest<HTMLElement>('[data-slot="setting-row"]')!);
}

function grant(...operations: number[]) {
  for (const operation of operations) {
    mocks.granted.add(`${Resource.Organization}:${operation}`);
  }
}

function renderForm(body: object = saved) {
  mocks.get.mockResolvedValue(body);
  mocks.put.mockImplementation(async (_endpoint: string, request: object) => ({
    ...body,
    ...request,
  }));
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <TestRouter>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={client}>
          <DataRetentionForm now={NOW} />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

describe("DataRetentionForm", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("groups the periods by where the data comes from, one tab and no tab row", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditAccessRetentionDays");

    expect(
      screen.getAllByRole("heading", { level: 2 }).map((heading) => heading.textContent),
    ).toEqual(["Audit trail", "EDI", "Driver files", "AI and agents"]);
    expect(screen.queryByRole("tab")).toBeNull();
  });

  it("says what the next purge removes and the date it reaches back to", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditAccessRetentionDays");

    expect(
      row("Reads and downloads").getByText(
        "Entries from before Jul 28, 2026 are permanently deleted in the next nightly purge.",
      ),
    ).toBeInTheDocument();
    expect(
      row("Financial changes").getByText(
        "Entries from before Oct 28, 2019 are permanently deleted in the next nightly purge.",
      ),
    ).toBeInTheDocument();
    expect(
      row("Blank payloads after").getByText(
        "Payloads from before Sep 11, 2026 are blanked in the next nightly purge, and those messages can no longer be replayed.",
      ),
    ).toBeInTheDocument();
    expect(
      row("AI feedback").getByText(
        "Feedback from before Oct 26, 2024 is permanently deleted in the next nightly purge.",
      ),
    ).toBeInTheDocument();
    expect(
      row("Delete cases after").getByText(
        "Cases captured before Oct 26, 2025 and their replays are permanently deleted in the next nightly purge.",
      ),
    ).toBeInTheDocument();
  });

  it("names the floor the server holds each period to", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditAccessRetentionDays");

    expect(
      row("Reads and downloads").getByText(
        "Who viewed, downloaded or exported a record. At least 90 days.",
      ),
    ).toBeInTheDocument();
    expect(
      row("Financial changes").getByText(
        "Changes to invoices, rates, accounting and settlements. At least 2,555 days, seven years.",
      ),
    ).toBeInTheDocument();
  });

  it("draws a purge that is off with its switch off and no period", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("ediMessageRetentionPeriod");

    expect(screen.getByRole("switch", { name: "Blank old inbound files" })).not.toBeChecked();
    expect(input("ediInboundFileRetentionPeriod")).toBeNull();
    expect(screen.queryByText(/Files received before/)).toBeNull();
    expect(screen.getByRole("switch", { name: "Blank old message payloads" })).toBeChecked();
    expect(input("ediMessageRetentionPeriod")).toHaveValue("45");
    expect(screen.getByRole("switch", { name: "Delete old evaluation cases" })).toBeChecked();
  });

  it("saves a purge turned on that never had a period, starting from 90 days", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    await field("ediMessageRetentionPeriod");
    await user.click(screen.getByRole("switch", { name: "Blank old inbound files" }));

    expect(await field("ediInboundFileRetentionPeriod")).toHaveValue("90");
    expect(
      row("Blank files after").getByText(
        "Files received before Jul 28, 2026 are blanked in the next nightly purge. This cannot be undone.",
      ),
    ).toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/data-retention/", {
        ...savedRequest,
        ediInboundFileRetentionPeriod: 90,
      }),
    );
  });

  it("saves 0 for a purge turned off and brings back its last period when turned on", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    await field("ediMessageRetentionPeriod");
    const purge = () => screen.getByRole("switch", { name: "Blank old message payloads" });

    await user.click(purge());
    await waitFor(() => expect(input("ediMessageRetentionPeriod")).toBeNull());

    await user.click(await screen.findByRole("button", { name: "Save changes" }));
    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/data-retention/", {
        ...savedRequest,
        ediMessageRetentionPeriod: 0,
      }),
    );

    await user.click(purge());

    expect(await field("ediMessageRetentionPeriod")).toHaveValue("45");
  });

  it("refuses an AI audit trail shorter than a year, before saving", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    const trail = await field("aiAuditRetentionPeriod");
    await user.clear(trail);
    await user.type(trail, "200");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(
      await screen.findByText("The AI audit trail is kept for at least 365 days"),
    ).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("refuses driver qualification files kept under three years, before saving", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    const files = await field("driverQualificationRetentionPeriod");
    await user.clear(files);
    await user.type(files, "1000");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(
      await screen.findByText(
        "Driver qualification files must be kept for at least 1,095 days (three years) after employment ends",
      ),
    ).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("asks for a number of days when a period is emptied, before saving", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    await user.clear(await field("aiCorrectionRetentionPeriod"));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(
      await screen.findByText("Enter how many days to keep AI corrections"),
    ).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("puts a period the server refuses on its own row", async () => {
    grant(Operation.Read, Operation.Update);
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    const user = userEvent.setup();
    renderForm();
    mocks.put.mockRejectedValue(
      new ApiRequestError(400, {
        type: "https://api.trenova.app/problems/validation-error",
        title: "Validation Failed",
        status: 400,
        detail: "Security entries are kept for at least 730 days",
        instance: "/data-retention/",
        code: "INVALID",
        errors: [
          {
            path: ["auditSecurityRetentionDays"],
            message: "Security entries are kept for at least 730 days",
            code: "INVALID",
            location: "body",
          },
        ],
      } as ConstructorParameters<typeof ApiRequestError>[1]),
    );

    const security = await field("auditSecurityRetentionDays");
    await user.clear(security);
    await user.type(security, "800");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(
        row("Security entries").getAllByText("Security entries are kept for at least 730 days")
          .length,
      ).toBeGreaterThan(0),
    );
  });

  it("shows the periods read-only to someone who may not update them", async () => {
    grant(Operation.Read);
    const user = userEvent.setup();
    renderForm();

    expect(
      await screen.findByText("You can view these settings but not change them."),
    ).toBeInTheDocument();
    expect(await field("auditAccessRetentionDays")).toBeDisabled();
    for (const toggle of screen.getAllByRole("switch")) {
      expect(toggle).toHaveAttribute("aria-disabled", "true");
    }

    await user.click(screen.getAllByRole("switch")[0]!);
    expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull();
    expect(mocks.put).not.toHaveBeenCalled();
  });
});

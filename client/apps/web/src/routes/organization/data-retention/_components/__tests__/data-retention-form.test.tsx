import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Suspense, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import DataRetentionForm from "../data-retention-form";

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
window of 0 on the EDI and evaluation-case fields means no limit.
*/
const saved = {
  id: "dr_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  auditRetentionPeriod: 120,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 45,
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

const savedRequest = {
  auditRetentionPeriod: 120,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 45,
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
  return within(screen.getByText(label).closest("tr")!);
}

function grant(...operations: number[]) {
  for (const operation of operations) {
    mocks.granted.add(`${Resource.Organization}:${operation}`);
  }
}

function renderForm(body: typeof saved = saved) {
  mocks.get.mockResolvedValue(body);
  mocks.put.mockImplementation(async (_endpoint: string, request: object) => ({
    ...body,
    ...request,
  }));
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <Suspense fallback={null}>{children}</Suspense>
    </QueryClientProvider>
  );
  return render(<DataRetentionForm now={NOW} />, { wrapper });
}

describe("DataRetentionForm", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("shows the date each window reaches back to and what happens to older data", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditRetentionPeriod");

    expect(row("Audit entries").getByText("Jun 28, 2026")).toBeInTheDocument();
    expect(row("Audit entries").getByText("Deleted")).toBeInTheDocument();
    expect(row("Message payloads").getByText("Sep 11, 2026")).toBeInTheDocument();
    expect(row("Message payloads").getByText("Raw content blanked")).toBeInTheDocument();
    expect(row("AI audit trail").getByText("Oct 28, 2019")).toBeInTheDocument();
    expect(row("AI feedback").getByText("Oct 26, 2024")).toBeInTheDocument();
    expect(row("Agent evaluation cases").getByText("Oct 26, 2025")).toBeInTheDocument();
    expect(row("Agent evaluation cases").getByText("Purged with replays")).toBeInTheDocument();
  });

  it("names the floor the server holds each window to", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditRetentionPeriod");

    expect(row("Audit entries").getByText("At least 1 day")).toBeInTheDocument();
    expect(row("AI audit trail").getByText("At least 365 days")).toBeInTheDocument();
    expect(row("AI feedback").getByText("At least 30 days")).toBeInTheDocument();
    expect(row("AI corrections").getByText("At least 30 days")).toBeInTheDocument();
  });

  it("shows a window kept forever as kept, with no cutoff", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("auditRetentionPeriod");

    expect(row("Inbound file contents").getByText("Kept")).toBeInTheDocument();
    expect(row("Inbound file contents").getByText("—")).toBeInTheDocument();
  });

  it("shows a window the server keeps forever as Keep forever, with no day count", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm();

    await field("ediMessageRetentionPeriod");
    const [inboundForever, messageForever] = screen.getAllByRole("switch", {
      name: "Keep forever",
    });

    expect(inboundForever).toBeChecked();
    expect(messageForever).not.toBeChecked();
    expect(input("ediInboundFileRetentionPeriod")).toBeNull();
    expect(input("ediMessageRetentionPeriod")).toHaveValue("45");
  });

  it("saves a limit put on a window that kept everything, starting from 90 days", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    await field("ediMessageRetentionPeriod");
    await user.click(screen.getAllByRole("switch", { name: "Keep forever" })[0]!);

    expect(await field("ediInboundFileRetentionPeriod")).toHaveValue("90");
    expect(row("Inbound file contents").getByText("Jul 28, 2026")).toBeInTheDocument();

    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/data-retention/", {
        ...savedRequest,
        ediInboundFileRetentionPeriod: 90,
      }),
    );
  });

  it("saves 0 for Keep forever and brings back the last limit when it is turned off", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm();

    await field("ediMessageRetentionPeriod");
    const messageForever = () => screen.getAllByRole("switch", { name: "Keep forever" })[1]!;

    await user.click(messageForever());
    expect(input("ediMessageRetentionPeriod")).toBeNull();

    await user.click(await screen.findByRole("button", { name: "Save changes" }));
    await waitFor(() =>
      expect(mocks.put).toHaveBeenCalledWith("/data-retention/", {
        ...savedRequest,
        ediMessageRetentionPeriod: 0,
      }),
    );

    await user.click(messageForever());

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
      await screen.findByText("AI audit trail retention must be at least 365 days"),
    ).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("shows the windows read-only to someone who may not update them", async () => {
    grant(Operation.Read);
    const user = userEvent.setup();
    renderForm();

    expect(
      await screen.findByText("You can view these settings but not change them."),
    ).toBeInTheDocument();
    expect(await field("auditRetentionPeriod")).toBeDisabled();
    for (const toggle of screen.getAllByRole("switch")) {
      expect(toggle).toHaveAttribute("aria-disabled", "true");
    }

    await user.click(screen.getAllByRole("switch")[0]!);
    await waitFor(() => expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull());
    expect(mocks.put).not.toHaveBeenCalled();
  });
});

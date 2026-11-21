import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { Suspense } from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import CaptureSettingsForm from "../_components/capture-settings-form";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/services/document-control", () => ({
  documentControlService: { get: mocks.get, update: mocks.update },
}));

vi.mock("@/lib/graphql/capture", () => ({
  fetchMyCaptureAccess: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

/** The record as `GET /document-controls/` sends it: the Go entity's JSON. */
const saved: DocumentControl = {
  id: "docc_1",
  version: 4,
  createdAt: 1_780_000_000,
  updatedAt: 1_780_000_000,
  organizationId: "org_1",
  businessUnitId: "bu_1",
  enableDocumentIntelligence: true,
  enableOcr: true,
  enableAutoClassification: true,
  enableAutoDocumentTypeAssociate: true,
  enableAutoCreateDocumentTypes: false,
  enableShipmentDraftExtraction: true,
  enableAiAssistedClassification: true,
  enableAiAssistedExtraction: true,
  shipmentDraftAllowedResources: ["shipment", "worker"],
  enableFullTextIndexing: true,
  enableCapture: true,
  captureAutoFileCoverSheets: true,
  captureRetentionDays: 30,
  captureMinAgentVersion: "",
  captureAllowAutoUpdate: true,
};

const RETENTION = "captureRetentionDays";
const MIN_VERSION = "captureMinAgentVersion";

async function field(name: string) {
  return waitFor(() => {
    const input = document.querySelector<HTMLInputElement>(`input[name="${name}"]`);
    expect(input).not.toBeNull();
    return input!;
  });
}

function grant(...operations: number[]) {
  for (const operation of operations) {
    mocks.granted.add(`${Resource.DocumentControl}:${operation}`);
  }
}

function renderForm(control: DocumentControl) {
  mocks.get.mockResolvedValue(control);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <TestRouter>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={client}>
          <Suspense fallback={null}>
            <CaptureSettingsForm />
          </Suspense>
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

describe("CaptureSettingsForm", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("draws the three sections as rows while capture is on", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm(saved);

    await field(RETENTION);
    expect(
      screen.getAllByRole("heading", { level: 2 }).map((heading) => heading.textContent),
    ).toEqual(["Availability", "Filing", "Desktop app"]);
    expect(document.querySelectorAll('[data-slot="setting-row"]')).toHaveLength(5);
  });

  it("hides filing and the desktop app while capture is off, and shows them when it is turned on", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm({ ...saved, enableCapture: false });

    await screen.findByRole("heading", { name: "Availability" });
    expect(screen.queryByRole("heading", { name: "Filing" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Desktop app" })).toBeNull();
    expect(document.querySelector(`input[name="${RETENTION}"]`)).toBeNull();

    await user.click(screen.getByRole("switch", { name: /Scanning and printing/ }));

    expect(await screen.findByRole("heading", { name: "Filing" })).toBeInTheDocument();
    expect(await field(RETENTION)).toBeEnabled();
  });

  it("sends the whole record, with the document extraction settings as they were read", async () => {
    grant(Operation.Read, Operation.Update);
    mocks.update.mockImplementation(async (values: DocumentControl) => values);
    const user = userEvent.setup();
    renderForm(saved);

    const retention = await field(RETENTION);
    await user.clear(retention);
    await user.type(retention, "45");
    await user.type(await field(MIN_VERSION), "1.4.0");
    await user.click(screen.getByRole("switch", { name: /Update on its own/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(mocks.update.mock.calls[0]![0]).toEqual({
      ...saved,
      captureRetentionDays: 45,
      captureMinAgentVersion: "1.4.0",
      captureAllowAutoUpdate: false,
    });
  });

  it("refuses a minimum version the server would refuse, before saving", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm(saved);

    await user.type(await field(MIN_VERSION), "1.4");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect((await screen.findAllByText("Use a version like 1.4.0")).length).toBeGreaterThan(0);
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it("puts the server's refusal on the field it names", async () => {
    grant(Operation.Read, Operation.Update);
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    mocks.update.mockRejectedValue(
      new ApiRequestError(400, {
        type: "https://api.trenova.app/problems/validation-error",
        title: "Validation Failed",
        status: 400,
        detail: "Intake retention must be at most 365 days",
        instance: "/document-controls/",
        code: "INVALID",
        errors: [
          {
            path: ["captureRetentionDays"],
            message: "Intake retention must be at most 365 days",
            code: "INVALID",
            location: "body",
          },
        ],
      } as ConstructorParameters<typeof ApiRequestError>[1]),
    );
    const user = userEvent.setup();
    renderForm(saved);

    await user.click(await screen.findByRole("switch", { name: /File behind cover sheets/ }));
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    const row = await waitFor(() => {
      const found = (document.querySelector(`input[name="${RETENTION}"]`) as HTMLElement).closest(
        '[data-slot="setting-row"]',
      ) as HTMLElement;
      expect(within(found).getByText("Intake retention must be at most 365 days")).toBeVisible();
      return found;
    });
    expect(within(row).getByText("Keep unfiled pages for")).toBeInTheDocument();
  });

  it("shows the settings held to someone who may not update them, with nothing to save", async () => {
    grant(Operation.Read);
    const user = userEvent.setup();
    renderForm(saved);

    expect(
      await screen.findByText("You can view these settings but not change them."),
    ).toBeInTheDocument();
    expect(await field(RETENTION)).toBeDisabled();
    expect(await field(MIN_VERSION)).toBeDisabled();
    for (const toggle of screen.getAllByRole("switch")) {
      expect(toggle).toHaveAttribute("aria-disabled", "true");
    }

    await user.click(screen.getAllByRole("switch")[0]!);
    await waitFor(() => expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull());
    expect(mocks.update).not.toHaveBeenCalled();
  });
});

describe("captureMinAgentVersion", () => {
  const parse = (captureMinAgentVersion: string) =>
    documentControlSchema.safeParse({ ...saved, captureMinAgentVersion }).success;

  it("accepts blank and what versionutils.Parse accepts", () => {
    expect(parse("")).toBe(true);
    expect(parse("1.4.0")).toBe(true);
    expect(parse("0.0.1")).toBe(true);
    expect(parse("v1.12.3")).toBe(true);
    expect(parse("10.20.30")).toBe(true);
  });

  it("refuses what versionutils.Parse refuses", () => {
    for (const raw of [
      "1",
      "1.2",
      "1.2.3.4",
      "1.02.3",
      "1.a.3",
      "1.-2.3",
      "1.2.3-beta",
      "latest",
    ]) {
      expect(parse(raw), raw).toBe(false);
    }
  });

  it("refuses a version longer than the column holds", () => {
    expect(parse("1234567890.123456789.1")).toBe(false);
  });
});

describe("captureRetentionDays", () => {
  const issue = (captureRetentionDays: number | null) =>
    documentControlSchema
      .safeParse({ ...saved, captureRetentionDays })
      .error?.issues.find((entry) => entry.path[0] === "captureRetentionDays")?.message;

  it("takes the server's bounds, 1 to 365 days, in words", () => {
    expect(issue(1)).toBeUndefined();
    expect(issue(365)).toBeUndefined();
    expect(issue(0)).toBe("Keep unfiled pages for at least 1 day");
    expect(issue(366)).toBe("Keep unfiled pages for at most 365 days");
    expect(issue(null)).toBe("Enter how many days to keep unfiled pages");
  });
});

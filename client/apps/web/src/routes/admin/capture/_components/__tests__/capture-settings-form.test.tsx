import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Suspense, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import CaptureSettingsForm from "../capture-settings-form";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/services/api", () => ({
  apiService: { documentControlService: { get: mocks.get, update: mocks.update } },
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

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }));

const saved: DocumentControl = {
  id: "dc_1",
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
  shipmentDraftAllowedResources: ["shipment"],
  enableFullTextIndexing: true,
  enableCapture: true,
  captureAutoFileCoverSheets: true,
  captureRetentionDays: 30,
  captureMinAgentVersion: "",
  captureAllowAutoUpdate: true,
};

const RETENTION = "captureRetentionDays";
const MIN_VERSION = "captureMinAgentVersion";

// The field components render their label beside, not for, the input.
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
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <Suspense fallback={null}>{children}</Suspense>
    </QueryClientProvider>
  );
  return render(<CaptureSettingsForm />, { wrapper });
}

describe("CaptureSettingsForm", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("holds retention with the other capture settings while capture is off", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm({ ...saved, enableCapture: false });

    expect(await field(RETENTION)).toBeDisabled();
  });

  it("lets someone who can update change retention while capture is on", async () => {
    grant(Operation.Read, Operation.Update);
    renderForm(saved);

    expect(await field(RETENTION)).toBeEnabled();
  });

  it("refuses a minimum version the server would refuse, before saving", async () => {
    grant(Operation.Read, Operation.Update);
    const user = userEvent.setup();
    renderForm(saved);

    await user.type(await field(MIN_VERSION), "1.4");
    await user.click(await screen.findByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("Use a version like 1.4.0")).toBeInTheDocument();
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it("shows the settings read-only to someone who may not update them", async () => {
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
});

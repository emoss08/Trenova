import type { CarrierIntelSettings } from "@/lib/graphql/carrier-intelligence";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Tabs } from "@trenova/shared/components/ui/tabs";
import { useController, type Control, type FieldValues } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CarrierIntelSettingsForm } from "../carrier-intel-settings-form";
import { requiresCostConfirmation } from "../carrier-intel-settings-schema";

const mocks = vi.hoisted(() => ({
  fetchCarrierIntelCostEstimate: vi.fn(),
  fetchCarrierIntelMonitoringStatus: vi.fn(),
  fetchCarrierIntelUsage: vi.fn(),
  updateCarrierIntelControl: vi.fn(),
  switchCarrierIntelProvider: vi.fn(),
  resumeCarrierIntelMonitoring: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intel-settings", () => mocks);

function SelectFieldStub({
  control,
  name,
  label,
  options,
}: {
  control: Control<FieldValues>;
  name: string;
  label: string;
  options: { value: string; label: string }[];
}) {
  const { field } = useController({ control, name });
  return (
    <label>
      {label}
      <select
        aria-label={label}
        value={field.value ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

vi.mock("@/components/fields/select-field", () => ({ SelectField: SelectFieldStub }));

const settings: CarrierIntelSettings = {
  carrierIntelControl: {
    id: "cictl_1",
    primaryProvider: "CarrierOK",
    fallbackProvider: null,
    enrollmentPolicy: "Manual",
    recentUsageDays: 90,
    includeOpenTenders: false,
    autoEnrollOnCreate: true,
    autoUnenrollOnInactive: true,
    exclusiveWatchlist: false,
    pollIntervalMinutes: 120,
    snapshotTtlHours: 24,
    fullProfileTtlDays: 30,
    preTenderRefreshEnabled: false,
    preTenderMaxAgeHours: 24,
    hardMaxAgeHours: 168,
    confirmBlockingChanges: true,
    outagePolicy: "FailOpen",
    autoDisqualifyOnBlock: false,
    autoApplySafetyRating: true,
    rules: [],
    autoSyncFields: [],
    monthlySpendCap: null,
    softCapPercent: 80,
    dailyFullProfileCap: null,
    rawRetentionDays: 90,
    snapshotHistoryLimit: 12,
    selfMonitoringEnabled: true,
    policyVersion: 3,
    version: 7,
    updatedAt: 1_757_000_000,
  },
  carrierIntelProvider: {
    configured: true,
    provider: "CarrierOK",
    fallbackProvider: null,
    capabilities: ["LookupFull", "NativeMonitoring"],
    sections: ["Identity", "Authority", "Insurance", "Safety"],
  },
  carrierIntelRuleCatalog: [],
};

const estimate = {
  provider: "CarrierOK",
  policy: "AllActive",
  subjectCount: 1250,
  monthlyMonitoring: "3125.00",
  perSubject: "2.50",
};

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <Tabs value="monitoring">
        <CarrierIntelSettingsForm
          settings={settings}
          open
          activeTab="monitoring"
          canManage
          onTabChange={vi.fn()}
          onClose={vi.fn()}
        />
      </Tabs>
    </QueryClientProvider>,
  );
}

describe("carrier intelligence cost confirmation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchCarrierIntelCostEstimate.mockResolvedValue(estimate);
    mocks.fetchCarrierIntelMonitoringStatus.mockResolvedValue({
      provider: settings.carrierIntelProvider,
      enrollmentCounts: { desired: 0, active: 0, pending: 0, failed: 0 },
      eventCounts: { open: 0, acknowledged: 0, bySeverity: [] },
      feeds: [],
      reviewQueueCount: 0,
    });
    mocks.updateCarrierIntelControl.mockImplementation(async (input) => ({
      ...settings.carrierIntelControl,
      enrollmentPolicy: input.enrollmentPolicy ?? settings.carrierIntelControl.enrollmentPolicy,
      exclusiveWatchlist:
        input.exclusiveWatchlist ?? settings.carrierIntelControl.exclusiveWatchlist,
      version: 8,
    }));
  });

  it("requires confirming the estimate before enrolling every active carrier", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Enrollment policy" }),
      "AllActive",
    );
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("Confirm monitoring cost")).toBeInTheDocument();
    expect(screen.getByTestId("cost-estimate-subjects")).toHaveTextContent("1,250");
    expect(screen.getByTestId("cost-estimate-monthly")).toHaveTextContent("$3,125.00");
    expect(mocks.fetchCarrierIntelCostEstimate).toHaveBeenCalledWith(
      { policy: "AllActive", recentUsageDays: undefined, includeOpenTenders: undefined },
      expect.anything(),
    );
    expect(mocks.updateCarrierIntelControl).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Confirm and save" }));

    await waitFor(() => expect(mocks.updateCarrierIntelControl).toHaveBeenCalledTimes(1));
    expect(mocks.updateCarrierIntelControl).toHaveBeenCalledWith({
      enrollmentPolicy: "AllActive",
      version: 7,
      confirmEstimatedCost: true,
    });
  });

  it("does not save when the estimate is dismissed", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.selectOptions(
      screen.getByRole("combobox", { name: "Enrollment policy" }),
      "AllActive",
    );
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await screen.findByText("Confirm monitoring cost");

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() =>
      expect(screen.queryByText("Confirm monitoring cost")).not.toBeInTheDocument(),
    );
    expect(mocks.updateCarrierIntelControl).not.toHaveBeenCalled();
  });

  it("saves changes that do not affect enrollment without an estimate", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("switch", { name: "Exclusive watchlist" }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.updateCarrierIntelControl).toHaveBeenCalledTimes(1));
    expect(mocks.updateCarrierIntelControl).toHaveBeenCalledWith({
      exclusiveWatchlist: true,
      version: 7,
    });
    expect(screen.queryByText("Confirm monitoring cost")).not.toBeInTheDocument();
    expect(mocks.fetchCarrierIntelCostEstimate).not.toHaveBeenCalled();
  });
});

describe("requiresCostConfirmation", () => {
  const manual = {
    enrollmentPolicy: "Manual",
    recentUsageDays: 90,
    includeOpenTenders: false,
  } as const;
  const recent = { ...manual, enrollmentPolicy: "RecentlyUsed" } as const;
  const allActive = { ...manual, enrollmentPolicy: "AllActive" } as const;

  it("asks when enrollment widens to every active carrier", () => {
    expect(requiresCostConfirmation(allActive, manual)).toBe(true);
    expect(requiresCostConfirmation(allActive, recent)).toBe(true);
    expect(requiresCostConfirmation(allActive, allActive)).toBe(false);
  });

  it("asks when the recently used window or tender rule changes", () => {
    expect(requiresCostConfirmation(recent, manual)).toBe(true);
    expect(requiresCostConfirmation({ ...recent, recentUsageDays: 180 }, recent)).toBe(true);
    expect(requiresCostConfirmation({ ...recent, includeOpenTenders: true }, recent)).toBe(true);
    expect(requiresCostConfirmation(recent, recent)).toBe(false);
  });

  it("never asks when narrowing to manual enrollment", () => {
    expect(requiresCostConfirmation(manual, allActive)).toBe(false);
    expect(requiresCostConfirmation(manual, recent)).toBe(false);
  });
});

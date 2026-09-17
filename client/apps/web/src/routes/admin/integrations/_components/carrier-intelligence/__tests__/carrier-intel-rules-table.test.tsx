import type {
  CarrierIntelControl,
  CarrierIntelRuleDefinition,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelSection } from "@trenova/graphql/generated/graphql";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import {
  buildControlPatch,
  remapRuleFieldPath,
  toSettingsFormValues,
  type CarrierIntelSettingsFormValues,
} from "../carrier-intel-settings-schema";
import { CarrierIntelRulesTable } from "../rules-tab";

const catalog: CarrierIntelRuleDefinition[] = [
  {
    code: "authority.usdot_inactive",
    label: "USDOT inactive",
    description: "The USDOT number is not active.",
    category: "Authority",
    defaultAction: "Block",
    recommendedAction: "Block",
    requiredSections: ["Authority"],
    subjects: ["Carrier"],
    gateRelevant: true,
    params: [],
  },
  {
    code: "safety.iss_high",
    label: "High inspection selection score",
    description: "The ISS score marks the carrier for inspection.",
    category: "Safety",
    defaultAction: "Warn",
    recommendedAction: "Warn",
    requiredSections: ["Safety"],
    subjects: ["Carrier"],
    gateRelevant: true,
    params: [
      {
        key: "minimum",
        label: "Minimum ISS score",
        type: "Integer",
        default: "75",
        min: 1,
        max: 100,
        options: null,
        helpText: null,
      },
    ],
  },
  {
    code: "fraud.network_sharing",
    label: "Shares identity with other carriers",
    description: "The carrier shares identifiers with other USDOT numbers.",
    category: "Network",
    defaultAction: "Warn",
    recommendedAction: "Warn",
    requiredSections: ["Network"],
    subjects: ["Carrier"],
    gateRelevant: true,
    params: [],
  },
];

const control: CarrierIntelControl = {
  id: "cictl_1",
  primaryProvider: "FMCSAQCMobile",
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
};

const fmcsaSections: CarrierIntelSection[] = ["Identity", "Authority", "Insurance", "Safety"];

function Harness({
  sections,
  onValues,
}: {
  sections: CarrierIntelSection[];
  onValues: (values: CarrierIntelSettingsFormValues) => void;
}) {
  const form = useForm<CarrierIntelSettingsFormValues>({
    defaultValues: toSettingsFormValues(control, catalog),
  });

  return (
    <FormProvider {...form}>
      <CarrierIntelRulesTable
        catalog={catalog}
        sections={sections}
        providerName="FMCSA QCMobile"
        providerConfigured
      />
      <button type="button" onClick={() => onValues(form.getValues())}>
        read
      </button>
    </FormProvider>
  );
}

describe("CarrierIntelRulesTable", () => {
  it("groups rules by category and marks rules the provider cannot evaluate", () => {
    render(<Harness sections={fmcsaSections} onValues={vi.fn()} />);

    expect(screen.getByText("Authority")).toBeInTheDocument();
    expect(screen.getByText("Network signals")).toBeInTheDocument();

    const unsupported = screen.getByTestId("carrier-intel-rule-fraud.network_sharing");
    expect(unsupported).toHaveAttribute("data-supported", "false");
    expect(within(unsupported).getByText("Not provided by FMCSA QCMobile")).toBeInTheDocument();

    const supported = screen.getByTestId("carrier-intel-rule-authority.usdot_inactive");
    expect(supported).toHaveAttribute("data-supported", "true");
    expect(within(supported).queryByText(/Not provided by/)).not.toBeInTheDocument();
  });

  it("shows every rule as supported when the provider returns the required sections", () => {
    render(<Harness sections={[...fmcsaSections, "Network"]} onValues={vi.fn()} />);

    expect(screen.queryByText(/Not provided by/)).not.toBeInTheDocument();
  });

  it("changes a rule action and sends only that rule in the patch", async () => {
    const user = userEvent.setup();
    const onValues = vi.fn();
    render(<Harness sections={fmcsaSections} onValues={onValues} />);

    await user.click(
      screen.getByRole("combobox", { name: "High inspection selection score action" }),
    );
    await user.click(await screen.findByRole("option", { name: "Block" }));
    await user.click(screen.getByText("read"));

    const values = onValues.mock.lastCall?.[0] as CarrierIntelSettingsFormValues;
    expect(values.rules[1]).toEqual({
      code: "safety.iss_high",
      action: "Block",
      params: { minimum: "75" },
    });

    const original = toSettingsFormValues(control, catalog);
    expect(buildControlPatch({ values, original, catalog, storedCodes: new Set() })).toEqual({
      version: 7,
      rules: [
        { code: "safety.iss_high", action: "Block", params: [{ key: "minimum", value: "75" }] },
      ],
    });
  });

  it("maps server rule errors onto the row that owns them", () => {
    const rules = toSettingsFormValues(control, catalog).rules;

    expect(remapRuleFieldPath("rules.safety.iss_high.params.minimum", rules)).toBe(
      "rules.1.params.minimum",
    );
    expect(remapRuleFieldPath("rules.authority.usdot_inactive.action", rules)).toBe(
      "rules.0.action",
    );
    expect(remapRuleFieldPath("rules.unknown.rule", rules)).toBe("rules.unknown.rule");
    expect(remapRuleFieldPath("hardMaxAgeHours", rules)).toBe("hardMaxAgeHours");
  });
});

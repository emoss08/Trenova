import type { CarrierEligibility, CarrierEligibilityFinding } from "@trenova/shared/types/shipment";

export type EligibilityItemSource = "record" | "intelligence";

export type EligibilityItem = {
  key: string;
  code: string | null;
  message: string;
  source: EligibilityItemSource;
  requiresOverride: boolean;
};

export type GroupedEligibility = {
  recordBlockers: EligibilityItem[];
  intelBlockers: EligibilityItem[];
  warnings: EligibilityItem[];
  advisories: EligibilityItem[];
};

const SYSTEM_INTEL_CODE_PREFIX = "intel.";

function normalizeSource(source: string): EligibilityItemSource {
  return source.trim().toLowerCase() === "intelligence" ? "intelligence" : "record";
}

function findingItem(finding: CarrierEligibilityFinding, index: number): EligibilityItem {
  return {
    key: `${finding.severity}-${finding.source}-${finding.code}-${index}`,
    code: finding.code || null,
    message: finding.message,
    source: normalizeSource(finding.source),
    requiresOverride: finding.requiresOverride,
  };
}

function messageItems(
  messages: readonly string[],
  prefix: string,
  source: EligibilityItemSource,
  requiresOverride: boolean,
): EligibilityItem[] {
  return messages.map((message, index) => ({
    key: `${prefix}-${index}-${message}`,
    code: null,
    message,
    source,
    requiresOverride,
  }));
}

export function groupCarrierEligibility(
  eligibility: Pick<CarrierEligibility, "blockers" | "warnings"> &
    Partial<Pick<CarrierEligibility, "advisories" | "findings">>,
): GroupedEligibility {
  const findings = eligibility.findings ?? [];
  const advisories = eligibility.advisories ?? [];

  if (findings.length === 0) {
    return {
      recordBlockers: messageItems(eligibility.blockers, "blocker", "record", false),
      intelBlockers: [],
      warnings: messageItems(eligibility.warnings, "warning", "record", true),
      advisories: messageItems(advisories, "advisory", "intelligence", false),
    };
  }

  const grouped: GroupedEligibility = {
    recordBlockers: [],
    intelBlockers: [],
    warnings: [],
    advisories: [],
  };

  findings.forEach((finding, index) => {
    const item = findingItem(finding, index);
    switch (finding.severity.trim().toLowerCase()) {
      case "blocker":
        if (item.source === "intelligence") {
          grouped.intelBlockers.push(item);
        } else {
          grouped.recordBlockers.push(item);
        }
        break;
      case "warning":
        grouped.warnings.push(item);
        break;
      case "advisory":
        grouped.advisories.push(item);
        break;
      default:
        break;
    }
  });

  return grouped;
}

export function hasEligibilityContent(grouped: GroupedEligibility): boolean {
  return (
    grouped.recordBlockers.length > 0 ||
    grouped.intelBlockers.length > 0 ||
    grouped.warnings.length > 0 ||
    grouped.advisories.length > 0
  );
}

export function isOverridableIntelBlocker(item: EligibilityItem): boolean {
  return (
    item.source === "intelligence" &&
    item.code !== null &&
    !item.code.startsWith(SYSTEM_INTEL_CODE_PREFIX)
  );
}

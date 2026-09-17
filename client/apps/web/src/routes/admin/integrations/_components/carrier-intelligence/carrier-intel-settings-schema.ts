import { isSectionCovered } from "@/lib/carrier-intelligence";
import type {
  CarrierIntelControl,
  CarrierIntelRuleDefinition,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelRuleParamDefinition } from "@/lib/graphql/carrier-intel-settings";
import type {
  CarrierIntelControlPatchInput,
  CarrierIntelEnrollmentPolicy,
  CarrierIntelOutagePolicy,
  CarrierIntelRuleAction,
  CarrierIntelRuleSettingInput,
  CarrierIntelSection,
  CarrierIntelSyncField,
} from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { isRecord, parseCommaSeparatedList } from "@trenova/shared/lib/utils";
import { compareDecimalStrings, isDecimalString } from "@trenova/shared/types/decimal";
import { z } from "zod";

export const CARRIER_INTEL_RULE_ACTIONS = [
  "Block",
  "Warn",
  "Notify",
  "Off",
] as const satisfies readonly CarrierIntelRuleAction[];

export const CARRIER_INTEL_ENROLLMENT_POLICIES = [
  "Manual",
  "RecentlyUsed",
  "AllActive",
] as const satisfies readonly CarrierIntelEnrollmentPolicy[];

export const CARRIER_INTEL_OUTAGE_POLICIES = [
  "FailOpen",
  "FailClosed",
] as const satisfies readonly CarrierIntelOutagePolicy[];

export const CARRIER_INTEL_SYNC_FIELDS = [
  "safetyRating",
  "mcNumber",
  "name",
  "dbaName",
  "addressLine1",
  "city",
  "postalCode",
  "phone",
  "email",
] as const satisfies readonly CarrierIntelSyncField[];

export const ruleActionChoices: ReadonlyArray<{ value: CarrierIntelRuleAction; label: string }> = [
  { value: "Block", label: "Block" },
  { value: "Warn", label: "Warn" },
  { value: "Notify", label: "Notify" },
  { value: "Off", label: "Off" },
];

export const enrollmentPolicyChoices: ReadonlyArray<{
  value: CarrierIntelEnrollmentPolicy;
  label: string;
}> = [
  { value: "Manual", label: "Manual" },
  { value: "RecentlyUsed", label: "Recently used carriers" },
  { value: "AllActive", label: "All active carriers" },
];

export const outagePolicyChoices: ReadonlyArray<{
  value: CarrierIntelOutagePolicy;
  label: string;
}> = [
  { value: "FailOpen", label: "Fail open (allow with a warning)" },
  { value: "FailClosed", label: "Fail closed (block until verified)" },
];

export const CARRIER_INTEL_SECTION_ORDER = [
  "Identity",
  "Authority",
  "Insurance",
  "Safety",
  "Basics",
  "Inspections",
  "Crashes",
  "Fleet",
  "Equipment",
  "Contacts",
  "Operations",
  "ChangeHistory",
  "Network",
  "Lanes",
  "Benchmarks",
] as const satisfies readonly CarrierIntelSection[];

export const capabilityLabels: Record<string, string> = {
  LookupFull: "Full profile lookup",
  LookupLite: "Lite lookup",
  LookupFMCSA: "FMCSA lookup",
  Search: "Carrier search",
  Autocomplete: "Autocomplete",
  NativeMonitoring: "Native monitoring",
  SnapshotMonitoring: "Snapshot monitoring",
  EquipmentLookup: "VIN lookup",
  NetworkSignals: "Fraud network signals",
  Lanes: "Lane history",
  BrokerAuthority: "Broker authority",
  InsuranceHistory: "Insurance history",
  RiskScore: "Risk score",
};

export type CarrierIntelRuleFormValue = {
  code: string;
  action: CarrierIntelRuleAction;
  params: Record<string, string>;
};

export type CarrierIntelSettingsFormValues = {
  version: number;
  enrollmentPolicy: CarrierIntelEnrollmentPolicy;
  recentUsageDays: number;
  includeOpenTenders: boolean;
  autoEnrollOnCreate: boolean;
  autoUnenrollOnInactive: boolean;
  exclusiveWatchlist: boolean;
  pollIntervalMinutes: number;
  snapshotTtlHours: number;
  fullProfileTtlDays: number;
  preTenderRefreshEnabled: boolean;
  preTenderMaxAgeHours: number;
  hardMaxAgeHours: number;
  confirmBlockingChanges: boolean;
  outagePolicy: CarrierIntelOutagePolicy;
  autoDisqualifyOnBlock: boolean;
  autoApplySafetyRating: boolean;
  autoSyncFields: CarrierIntelSyncField[] | null;
  monthlySpendCap: string | null;
  softCapPercent: number;
  dailyFullProfileCap: number | null;
  rawRetentionDays: number;
  snapshotHistoryLimit: number;
  selfMonitoringEnabled: boolean;
  rules: CarrierIntelRuleFormValue[];
};

type ScalarSettingKey = Exclude<
  keyof CarrierIntelSettingsFormValues,
  "version" | "rules" | "autoSyncFields"
>;

const SCALAR_SETTING_KEYS = [
  "enrollmentPolicy",
  "recentUsageDays",
  "includeOpenTenders",
  "autoEnrollOnCreate",
  "autoUnenrollOnInactive",
  "exclusiveWatchlist",
  "pollIntervalMinutes",
  "snapshotTtlHours",
  "fullProfileTtlDays",
  "preTenderRefreshEnabled",
  "preTenderMaxAgeHours",
  "hardMaxAgeHours",
  "confirmBlockingChanges",
  "outagePolicy",
  "autoDisqualifyOnBlock",
  "autoApplySafetyRating",
  "monthlySpendCap",
  "softCapPercent",
  "dailyFullProfileCap",
  "rawRetentionDays",
  "snapshotHistoryLimit",
  "selfMonitoringEnabled",
] as const satisfies readonly ScalarSettingKey[];

function defaultRuleValue(rule: CarrierIntelRuleDefinition): CarrierIntelRuleFormValue {
  const params: Record<string, string> = {};
  for (const param of rule.params) {
    params[param.key] = param.default;
  }
  return { code: rule.code, action: rule.defaultAction, params };
}

export function toSettingsFormValues(
  control: CarrierIntelControl,
  catalog: readonly CarrierIntelRuleDefinition[],
): CarrierIntelSettingsFormValues {
  const storedByCode = new Map(control.rules.map((rule) => [rule.code, rule]));

  return {
    version: control.version,
    enrollmentPolicy: control.enrollmentPolicy,
    recentUsageDays: control.recentUsageDays,
    includeOpenTenders: control.includeOpenTenders,
    autoEnrollOnCreate: control.autoEnrollOnCreate,
    autoUnenrollOnInactive: control.autoUnenrollOnInactive,
    exclusiveWatchlist: control.exclusiveWatchlist,
    pollIntervalMinutes: control.pollIntervalMinutes,
    snapshotTtlHours: control.snapshotTtlHours,
    fullProfileTtlDays: control.fullProfileTtlDays,
    preTenderRefreshEnabled: control.preTenderRefreshEnabled,
    preTenderMaxAgeHours: control.preTenderMaxAgeHours,
    hardMaxAgeHours: control.hardMaxAgeHours,
    confirmBlockingChanges: control.confirmBlockingChanges,
    outagePolicy: control.outagePolicy,
    autoDisqualifyOnBlock: control.autoDisqualifyOnBlock,
    autoApplySafetyRating: control.autoApplySafetyRating,
    autoSyncFields: [...control.autoSyncFields],
    monthlySpendCap: control.monthlySpendCap,
    softCapPercent: control.softCapPercent,
    dailyFullProfileCap: control.dailyFullProfileCap,
    rawRetentionDays: control.rawRetentionDays,
    snapshotHistoryLimit: control.snapshotHistoryLimit,
    selfMonitoringEnabled: control.selfMonitoringEnabled,
    rules: catalog.map((rule) => {
      const base = defaultRuleValue(rule);
      const stored = storedByCode.get(rule.code);
      if (!stored) {
        return base;
      }
      const params = { ...base.params };
      for (const param of stored.params) {
        params[param.key] = param.value;
      }
      return { code: rule.code, action: stored.action, params };
    }),
  };
}

export function isRuleSupported(
  rule: Pick<CarrierIntelRuleDefinition, "requiredSections">,
  sections: readonly CarrierIntelSection[],
): boolean {
  return rule.requiredSections.every((section) => isSectionCovered(sections, section));
}

const INTEGER_PATTERN = /^-?\d+$/;

export function validateRuleParam(
  param: CarrierIntelRuleParamDefinition,
  raw: string,
): string | null {
  const value = raw.trim();
  if (value === "") {
    return param.type === "MultiSelect"
      ? translate("Select at least one option for {0}", param.label)
      : null;
  }

  let numeric: number;
  switch (param.type) {
    case "Integer":
      if (!INTEGER_PATTERN.test(value)) {
        return translate("{0} must be a whole number", param.label);
      }
      numeric = Number(value);
      break;
    case "Decimal":
      if (!isDecimalString(value)) {
        return translate("{0} must be a number", param.label);
      }
      numeric = Number(value);
      break;
    case "Number":
      numeric = Number(value);
      if (!Number.isFinite(numeric)) {
        return translate("{0} must be a number", param.label);
      }
      break;
    case "Select":
      return (param.options ?? []).includes(value)
        ? null
        : translate("{0} has an invalid option", param.label);
    case "MultiSelect": {
      const options = param.options ?? [];
      return parseCommaSeparatedList(value).every((item) => options.includes(item))
        ? null
        : translate("{0} has an invalid option", param.label);
    }
    default:
      return null;
  }

  if (param.min !== null && numeric < param.min) {
    return translate("{0} must be at least {1}", param.label, param.min);
  }
  if (param.max !== null && numeric > param.max) {
    return translate("{0} cannot exceed {1}", param.label, param.max);
  }
  return null;
}

function boundedInt(label: string, min: number, max: number) {
  return z
    .number({ error: translate("{0} is required", label) })
    .int(translate("{0} must be a whole number", label))
    .min(min, translate("{0} must be at least {1}", label, min))
    .max(max, translate("{0} cannot exceed {1}", label, max));
}

export function buildSettingsSchema(catalog: readonly CarrierIntelRuleDefinition[]) {
  const catalogByCode = new Map(catalog.map((rule) => [rule.code, rule]));

  return z
    .object({
      version: z.number(),
      enrollmentPolicy: z.enum(CARRIER_INTEL_ENROLLMENT_POLICIES),
      recentUsageDays: boundedInt("Recent usage window", 7, 365),
      includeOpenTenders: z.boolean(),
      autoEnrollOnCreate: z.boolean(),
      autoUnenrollOnInactive: z.boolean(),
      exclusiveWatchlist: z.boolean(),
      pollIntervalMinutes: boundedInt("Poll interval", 60, 1440),
      snapshotTtlHours: boundedInt("Snapshot freshness", 1, 720),
      fullProfileTtlDays: boundedInt("Full profile freshness", 1, 365),
      preTenderRefreshEnabled: z.boolean(),
      preTenderMaxAgeHours: boundedInt("Pre-tender freshness", 1, 720),
      hardMaxAgeHours: boundedInt("Maximum intelligence age", 1, 8760),
      confirmBlockingChanges: z.boolean(),
      outagePolicy: z.enum(CARRIER_INTEL_OUTAGE_POLICIES),
      autoDisqualifyOnBlock: z.boolean(),
      autoApplySafetyRating: z.boolean(),
      autoSyncFields: z.array(z.enum(CARRIER_INTEL_SYNC_FIELDS)).nullable(),
      monthlySpendCap: z
        .string()
        .nullable()
        .refine(
          (value) =>
            value === null ||
            value.trim() === "" ||
            (isDecimalString(value) && compareDecimalStrings(value, "0") >= 0),
          translate("Monthly spend cap must be a non-negative amount"),
        ),
      softCapPercent: boundedInt("Soft cap percent", 1, 100),
      dailyFullProfileCap: z
        .number()
        .int(translate("Daily full profile cap must be a whole number"))
        .min(0, translate("Daily full profile cap cannot be negative"))
        .nullable(),
      rawRetentionDays: boundedInt("Raw payload retention", 7, 730),
      snapshotHistoryLimit: boundedInt("Snapshot history", 1, 100),
      selfMonitoringEnabled: z.boolean(),
      rules: z.array(
        z.object({
          code: z.string(),
          action: z.enum(CARRIER_INTEL_RULE_ACTIONS),
          params: z.record(z.string(), z.string()),
        }),
      ),
    })
    .superRefine((values, ctx) => {
      if (values.hardMaxAgeHours < values.preTenderMaxAgeHours) {
        ctx.addIssue({
          code: "custom",
          path: ["hardMaxAgeHours"],
          message: translate(
            "Maximum intelligence age cannot be shorter than the pre-tender freshness window",
          ),
        });
      }
      values.rules.forEach((rule, index) => {
        const definition = catalogByCode.get(rule.code);
        if (!definition) {
          return;
        }
        for (const param of definition.params) {
          const message = validateRuleParam(param, rule.params[param.key] ?? "");
          if (message) {
            ctx.addIssue({
              code: "custom",
              path: ["rules", index, "params", param.key],
              message,
            });
          }
        }
      });
    });
}

function sameRule(left: CarrierIntelRuleFormValue, right: CarrierIntelRuleFormValue): boolean {
  if (left.action !== right.action) {
    return false;
  }
  const keys = new Set([...Object.keys(left.params), ...Object.keys(right.params)]);
  for (const key of keys) {
    if ((left.params[key] ?? "").trim() !== (right.params[key] ?? "").trim()) {
      return false;
    }
  }
  return true;
}

function sameSyncFields(
  leftValue: readonly string[] | null,
  rightValue: readonly string[] | null,
): boolean {
  const left = leftValue ?? [];
  const right = rightValue ?? [];
  if (left.length !== right.length) {
    return false;
  }
  const set = new Set(left);
  return right.every((field) => set.has(field));
}

function normalizeSpendCap(value: string | null): string | null {
  const trimmed = value?.trim() ?? "";
  return trimmed === "" ? null : trimmed;
}

function sameSpendCap(left: string | null, right: string | null): boolean {
  const a = normalizeSpendCap(left);
  const b = normalizeSpendCap(right);
  if (a === null || b === null) {
    return a === b;
  }
  return isDecimalString(a) && isDecimalString(b) ? compareDecimalStrings(a, b) === 0 : a === b;
}

export function buildRuleSettingsInput(
  values: readonly CarrierIntelRuleFormValue[],
  catalog: readonly CarrierIntelRuleDefinition[],
  storedCodes: ReadonlySet<string>,
): CarrierIntelRuleSettingInput[] {
  const catalogByCode = new Map(catalog.map((rule) => [rule.code, rule]));
  const input: CarrierIntelRuleSettingInput[] = [];

  for (const rule of values) {
    const definition = catalogByCode.get(rule.code);
    if (!definition) {
      continue;
    }
    if (!storedCodes.has(rule.code) && sameRule(rule, defaultRuleValue(definition))) {
      continue;
    }
    const params = definition.params
      .map((param) => ({ key: param.key, value: (rule.params[param.key] ?? "").trim() }))
      .filter((param) => param.value !== "");
    input.push({ code: rule.code, action: rule.action, params });
  }

  return input;
}

export function buildControlPatch({
  values,
  original,
  catalog,
  storedCodes,
}: {
  values: CarrierIntelSettingsFormValues;
  original: CarrierIntelSettingsFormValues;
  catalog: readonly CarrierIntelRuleDefinition[];
  storedCodes: ReadonlySet<string>;
}): CarrierIntelControlPatchInput | null {
  const patch: CarrierIntelControlPatchInput = {};
  let changed = false;

  for (const key of SCALAR_SETTING_KEYS) {
    if (key === "monthlySpendCap") {
      if (!sameSpendCap(values.monthlySpendCap, original.monthlySpendCap)) {
        patch.monthlySpendCap = normalizeSpendCap(values.monthlySpendCap);
        changed = true;
      }
      continue;
    }
    if (values[key] !== original[key]) {
      Object.assign(patch, { [key]: values[key] });
      changed = true;
    }
  }

  if (!sameSyncFields(values.autoSyncFields, original.autoSyncFields)) {
    patch.autoSyncFields = [...(values.autoSyncFields ?? [])];
    changed = true;
  }

  const rulesChanged =
    values.rules.length !== original.rules.length ||
    values.rules.some((rule, index) => !sameRule(rule, original.rules[index]));
  if (rulesChanged) {
    patch.rules = buildRuleSettingsInput(values.rules, catalog, storedCodes);
    changed = true;
  }

  if (!changed) {
    return null;
  }

  patch.version = values.version;
  return patch;
}

export function requiresCostConfirmation(
  values: Pick<
    CarrierIntelSettingsFormValues,
    "enrollmentPolicy" | "recentUsageDays" | "includeOpenTenders"
  >,
  original: Pick<
    CarrierIntelSettingsFormValues,
    "enrollmentPolicy" | "recentUsageDays" | "includeOpenTenders"
  >,
): boolean {
  if (values.enrollmentPolicy === "AllActive") {
    return original.enrollmentPolicy !== "AllActive";
  }
  if (values.enrollmentPolicy === "RecentlyUsed") {
    return (
      original.enrollmentPolicy !== "RecentlyUsed" ||
      values.recentUsageDays !== original.recentUsageDays ||
      values.includeOpenTenders !== original.includeOpenTenders
    );
  }
  return false;
}

export function isCostConfirmationRequiredError(error: unknown): boolean {
  if (!(error instanceof GraphQLRequestError)) {
    return false;
  }
  return error.graphQLErrors.some(
    (graphQLError) =>
      isRecord(graphQLError.params) &&
      String(graphQLError.params.requiresCostConfirmation) === "true",
  );
}

export function remapRuleFieldPath(field: string, rules: readonly { code: string }[]): string {
  if (!field.startsWith("rules.")) {
    return field;
  }
  const rest = field.slice("rules.".length);
  let matchIndex = -1;
  let matchLength = 0;
  for (let index = 0; index < rules.length; index++) {
    const code = rules[index].code;
    if ((rest === code || rest.startsWith(`${code}.`)) && code.length > matchLength) {
      matchIndex = index;
      matchLength = code.length;
    }
  }
  if (matchIndex === -1) {
    return field;
  }
  const suffix = rest.slice(matchLength);
  return suffix === "" ? `rules.${matchIndex}.action` : `rules.${matchIndex}${suffix}`;
}

export function remapRuleFieldErrors(error: unknown, rules: readonly { code: string }[]): unknown {
  if (!(error instanceof GraphQLRequestError) || error.getFieldErrors().length === 0) {
    return error;
  }

  return new GraphQLRequestError({
    kind: error.kind,
    message: error.message,
    status: error.status,
    graphQLErrors: error.graphQLErrors.map((graphQLError) => {
      if (!Array.isArray(graphQLError.errors)) {
        return graphQLError;
      }
      const errors = graphQLError.errors.map((entry: unknown) =>
        isRecord(entry) && typeof entry.field === "string"
          ? { ...entry, field: remapRuleFieldPath(entry.field, rules) }
          : entry,
      );
      return {
        ...graphQLError,
        errors,
        extensions: { ...graphQLError.extensions, errors },
      };
    }),
  });
}

export type CarrierIntelSettingsTab = "rules" | "monitoring" | "spend";

const settingsTabByField: Record<string, CarrierIntelSettingsTab> = {
  rules: "rules",
  outagePolicy: "rules",
  hardMaxAgeHours: "rules",
  preTenderRefreshEnabled: "rules",
  preTenderMaxAgeHours: "rules",
  confirmBlockingChanges: "rules",
  autoDisqualifyOnBlock: "rules",
  autoApplySafetyRating: "rules",
  autoSyncFields: "rules",
  enrollmentPolicy: "monitoring",
  recentUsageDays: "monitoring",
  includeOpenTenders: "monitoring",
  autoEnrollOnCreate: "monitoring",
  autoUnenrollOnInactive: "monitoring",
  exclusiveWatchlist: "monitoring",
  pollIntervalMinutes: "monitoring",
  snapshotTtlHours: "monitoring",
  fullProfileTtlDays: "monitoring",
  selfMonitoringEnabled: "monitoring",
  monthlySpendCap: "spend",
  softCapPercent: "spend",
  dailyFullProfileCap: "spend",
  rawRetentionDays: "spend",
  snapshotHistoryLimit: "spend",
};

export function settingsTabForErrors(
  errors: Partial<Record<string, unknown>>,
): CarrierIntelSettingsTab | null {
  for (const key of Object.keys(errors)) {
    const tab = settingsTabByField[key];
    if (tab) {
      return tab;
    }
  }
  return null;
}

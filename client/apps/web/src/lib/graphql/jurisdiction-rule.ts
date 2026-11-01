import {
  CreateJurisdictionRuleDocument,
  CreateJurisdictionRuleOverrideDocument,
  DeleteJurisdictionRuleOverrideDocument,
  UpdateJurisdictionRuleDocument,
  UpdateJurisdictionRuleOverrideDocument,
  VerifyJurisdictionRuleDocument,
  type JurisdictionRuleInput,
  type JurisdictionRuleOverrideInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  JurisdictionRule,
  VerifyJurisdictionRuleInput as VerifyJurisdictionRuleValues,
} from "@/types/jurisdiction-rule";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";

function decimalInput(value: number | null | undefined): string | null {
  return value === null || value === undefined ? null : String(value);
}

/** The statutory limits as the API takes them; verification is earned through verify. */
export function toJurisdictionRuleInput(values: JurisdictionRule): JurisdictionRuleInput {
  return {
    stateId: values.stateId,
    status: values.status,
    maxWidthFeet: values.maxWidthFeet,
    maxHeightFeet: values.maxHeightFeet,
    maxLengthFeet: values.maxLengthFeet,
    maxWeightPounds: values.maxWeightPounds,
    superloadWidthFeet: values.superloadWidthFeet ?? null,
    superloadWeightPounds: values.superloadWeightPounds ?? null,
    daylightOnly: values.daylightOnly,
    rushHourRestricted: values.rushHourRestricted,
    weekendRestricted: values.weekendRestricted,
    holidayRestricted: values.holidayRestricted,
    permitLeadTimeDays: values.permitLeadTimeDays,
    permitValidityDays: values.permitValidityDays,
    permitBaseFee: decimalInput(values.permitBaseFee),
    permitPerMileFee: decimalInput(values.permitPerMileFee),
    sourceNote: values.sourceNote ?? null,
    sourceUrl: values.sourceUrl ?? null,
    effectiveStartDate: values.effectiveStartDate ?? null,
    effectiveEndDate: values.effectiveEndDate ?? null,
    version: values.version ?? null,
  };
}

export function toJurisdictionRuleOverrideInput(
  values: JurisdictionRuleOverride,
): JurisdictionRuleOverrideInput {
  return {
    stateId: values.stateId,
    maxWidthFeet: values.maxWidthFeet ?? null,
    maxHeightFeet: values.maxHeightFeet ?? null,
    maxLengthFeet: values.maxLengthFeet ?? null,
    maxWeightPounds: values.maxWeightPounds ?? null,
    permitLeadTimeDays: values.permitLeadTimeDays ?? null,
    daylightOnly: values.daylightOnly ?? null,
    holidayRestricted: values.holidayRestricted ?? null,
    reason: values.reason,
    version: values.version ?? null,
  };
}

export async function createJurisdictionRule(values: JurisdictionRule) {
  const data = await requestGraphQL({
    document: CreateJurisdictionRuleDocument,
    operationName: "CreateJurisdictionRule",
    variables: { input: toJurisdictionRuleInput(values) },
  });
  return data.createJurisdictionRule;
}

export async function updateJurisdictionRule(id: string, values: JurisdictionRule) {
  const data = await requestGraphQL({
    document: UpdateJurisdictionRuleDocument,
    operationName: "UpdateJurisdictionRule",
    variables: { id, input: toJurisdictionRuleInput(values) },
  });
  return data.updateJurisdictionRule;
}

export async function verifyJurisdictionRule(id: string, values: VerifyJurisdictionRuleValues) {
  const data = await requestGraphQL({
    document: VerifyJurisdictionRuleDocument,
    operationName: "VerifyJurisdictionRule",
    variables: {
      id,
      input: {
        verificationState: values.verificationState,
        sourceNote: values.sourceNote ?? null,
        sourceUrl: values.sourceUrl ?? null,
      },
    },
  });
  return data.verifyJurisdictionRule;
}

export async function createJurisdictionRuleOverride(values: JurisdictionRuleOverride) {
  const data = await requestGraphQL({
    document: CreateJurisdictionRuleOverrideDocument,
    operationName: "CreateJurisdictionRuleOverride",
    variables: { input: toJurisdictionRuleOverrideInput(values) },
  });
  return data.createJurisdictionRuleOverride;
}

export async function updateJurisdictionRuleOverride(id: string, values: JurisdictionRuleOverride) {
  const data = await requestGraphQL({
    document: UpdateJurisdictionRuleOverrideDocument,
    operationName: "UpdateJurisdictionRuleOverride",
    variables: { id, input: toJurisdictionRuleOverrideInput(values) },
  });
  return data.updateJurisdictionRuleOverride;
}

export async function deleteJurisdictionRuleOverride(id: string): Promise<void> {
  await requestGraphQL({
    document: DeleteJurisdictionRuleOverrideDocument,
    operationName: "DeleteJurisdictionRuleOverride",
    variables: { id },
  });
}

import { isRecord } from "@trenova/shared/lib/utils";

export const QUOTA_EXCEEDED_CODE = "QUOTA_EXCEEDED";
export const PLAN_RESTRICTED_CODE = "PLAN_RESTRICTED";

const QUOTA_EXCEEDED_TYPE = "quota-exceeded";
const PLAN_RESTRICTED_TYPE = "plan-restricted";
const PAYMENT_REQUIRED_STATUS = 402;

export const SUBSCRIPTION_READ_ONLY_REASON = "subscription_read_only";
export const SIGNUPS_PAUSED_REASON = "signups_paused";

export type QuotaExceededNotice = {
  kind: "quota";
  meter: string;
  limit: number | null;
  used: number | null;
  plan: string;
  message: string;
};

export type PlanRestrictedNotice = {
  kind: "restricted";
  capability: string;
  reason: string;
  plan: string;
  message: string;
};

export type PlanLimitNotice = QuotaExceededNotice | PlanRestrictedNotice;

type PlanLimitSource = {
  code?: unknown;
  type?: unknown;
  status?: unknown;
  params?: unknown;
  message?: unknown;
};

function stringOf(value: unknown): string {
  if (typeof value === "string") {
    return value.trim();
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  return "";
}

function numberOf(value: unknown): number | null {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : null;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

function problemTypeSuffix(type: unknown): string {
  const raw = stringOf(type);
  if (raw === "") {
    return "";
  }
  return raw.split("/").pop() ?? "";
}

/**
 * Reads a plan-limit refusal out of one error payload: a REST problem body, or the
 * `extensions` of a GraphQL error with its message folded in. The code is the
 * contract; the problem type and a bare 402 are accepted too, so a refusal is still
 * recognised when a proxy or an older server drops one of them.
 */
export function planLimitFromSource(source: PlanLimitSource): PlanLimitNotice | null {
  const code = stringOf(source.code).toUpperCase();
  const type = problemTypeSuffix(source.type);
  const params = isRecord(source.params) ? source.params : {};
  const message = stringOf(source.message);

  const isQuota =
    code === QUOTA_EXCEEDED_CODE ||
    type === QUOTA_EXCEEDED_TYPE ||
    (code === "" && type !== PLAN_RESTRICTED_TYPE && source.status === PAYMENT_REQUIRED_STATUS);
  if (isQuota) {
    return {
      kind: "quota",
      meter: stringOf(params.meter),
      limit: numberOf(params.limit),
      used: numberOf(params.used),
      plan: stringOf(params.plan),
      message,
    };
  }

  if (code === PLAN_RESTRICTED_CODE || type === PLAN_RESTRICTED_TYPE) {
    return {
      kind: "restricted",
      capability: stringOf(params.capability),
      reason: stringOf(params.reason),
      plan: stringOf(params.plan),
      message,
    };
  }

  return null;
}

/** A REST problem response, as `ApiRequestError` carries it. */
export function planLimitFromProblem(status: number, data: unknown): PlanLimitNotice | null {
  if (!isRecord(data)) {
    return status === PAYMENT_REQUIRED_STATUS ? planLimitFromSource({ status }) : null;
  }

  return planLimitFromSource({
    code: data.code,
    type: data.type,
    status,
    params: data.params,
    message: data.detail ?? data.title,
  });
}

type GraphQLErrorLike = {
  code?: unknown;
  type?: unknown;
  params?: unknown;
  message?: unknown;
  extensions?: unknown;
};

/** The first plan-limit refusal among a GraphQL response's errors. */
export function planLimitFromGraphQLErrors(
  errors: readonly GraphQLErrorLike[],
  status?: number,
): PlanLimitNotice | null {
  for (const error of errors) {
    const extensions = isRecord(error.extensions) ? error.extensions : {};
    const notice = planLimitFromSource({
      code: error.code ?? extensions.code,
      type: error.type ?? extensions.type,
      params: error.params ?? extensions.params,
      message: error.message,
    });
    if (notice) {
      return notice;
    }
  }

  return status === PAYMENT_REQUIRED_STATUS ? planLimitFromSource({ status }) : null;
}

/**
 * Recognises a thrown request error without importing the classes that throw them,
 * which would make this module and the transports import each other. An
 * `ApiRequestError` carries `status` and the problem `data`; a `GraphQLRequestError`
 * carries `graphQLErrors`.
 */
export function planLimitFromError(error: unknown): PlanLimitNotice | null {
  if (!isRecord(error)) {
    return null;
  }

  if (Array.isArray(error.graphQLErrors)) {
    return planLimitFromGraphQLErrors(
      error.graphQLErrors as GraphQLErrorLike[],
      typeof error.status === "number" ? error.status : undefined,
    );
  }

  if (typeof error.status === "number" && "data" in error) {
    return planLimitFromProblem(error.status, error.data);
  }

  return null;
}

/** Returns whether the notice was shown; a handler may decline one it has no use for. */
export type PlanLimitHandler = (notice: PlanLimitNotice) => boolean;

let planLimitHandler: PlanLimitHandler | null = null;
const explainedErrors = new WeakSet<object>();

/**
 * The transports sit below the router and the dialog, so they can only signal. The
 * app registers the one handler that opens the plan-limit dialog; without one a
 * refusal still throws to its caller and is reported the ordinary way.
 */
export function setPlanLimitHandler(handler: PlanLimitHandler | null): void {
  planLimitHandler = handler;
}

/**
 * Called by a transport with the error it is about to throw. When the handler shows
 * the refusal, the error is remembered so the caller's own error reporting can stay
 * quiet instead of repeating it as a generic toast.
 */
export function reportPlanLimit(error: object): boolean {
  const notice = planLimitFromError(error);
  if (!notice || !planLimitHandler) {
    return false;
  }

  let shown = false;
  try {
    shown = planLimitHandler(notice);
  } catch (handlerError) {
    console.error("[plan-limit] handler failed", handlerError);
  }
  if (shown) {
    explainedErrors.add(error);
  }
  return shown;
}

/** Whether the plan-limit dialog has already explained this error to the person. */
export function isExplainedPlanLimitError(error: unknown): boolean {
  return typeof error === "object" && error !== null && explainedErrors.has(error);
}

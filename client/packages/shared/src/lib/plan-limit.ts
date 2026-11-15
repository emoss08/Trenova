import { markExplained } from "@trenova/shared/lib/explained-errors";
import { isRecord } from "@trenova/shared/lib/utils";
import { RouteRefusal } from "@trenova/shared/lib/router/refusal";

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
  feature: string;
  addOn: string;
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
      feature: stringOf(params.feature),
      addOn: stringOf(params.addOn),
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

/**
 * The add-on refusal an error carries: a `PLAN_RESTRICTED` that names the feature the plan
 * leaves out. A read-only workspace, an expired trial or the signup wait list names none and
 * is not one.
 */
export type PlanRestriction = {
  /** The feature key; never shown. */
  feature: string;
  /** The add-on's name as the server gave it; empty for a feature of the base plan. */
  addOn: string;
};

export function planRestrictionOfNotice(notice: PlanLimitNotice | null): PlanRestriction | null {
  if (!notice || notice.kind !== "restricted" || notice.feature === "") return null;
  return { feature: notice.feature, addOn: notice.addOn };
}

/** The add-on refusal a thrown request error (either transport) carries, or null. */
export function planRestrictionOf(error: unknown): PlanRestriction | null {
  return planRestrictionOfNotice(planLimitFromError(error));
}

/**
 * What a route gate throws for a page whose feature the organization's plan leaves out. It
 * carries the body the API sends for the same refusal, so the route's error component
 * explains it exactly as it explains a refused read.
 */
export function planRestrictedRefusal(restriction: PlanRestriction): RouteRefusal {
  return new RouteRefusal({
    status: 403,
    // i18n-ignore: the HTTP status text, never shown
    statusText: "Plan Restricted",
    data: {
      type: PLAN_RESTRICTED_TYPE,
      // i18n-ignore: the problem title the API sends, never shown; the error states name the add-on
      title: "Plan Restricted",
      status: 403,
      code: PLAN_RESTRICTED_CODE,
      params: { feature: restriction.feature, addOn: restriction.addOn },
    },
  });
}

/**
 * What the request that was refused did: a read (a GET, a GraphQL query, an event stream) or
 * a write. The handler decides from it: the web app leaves an add-on refusal of a read to the
 * page that asked, whose error state names the add-on, and explains a write in the dialog.
 */
export type PlanLimitRequest = "read" | "write";

/**
 * Returns whether the notice was shown, which marks the error explained so the caller does not
 * repeat it; a handler declines what it leaves to the caller.
 */
export type PlanLimitHandler = (notice: PlanLimitNotice, request: PlanLimitRequest) => boolean;

let planLimitHandler: PlanLimitHandler | null = null;

/**
 * The transports sit below the router and the dialog, so they can only signal. The
 * app registers the one handler that opens the plan-limit dialog; without one a
 * refusal still throws to its caller and is reported the ordinary way.
 */
export function setPlanLimitHandler(handler: PlanLimitHandler | null): void {
  planLimitHandler = handler;
}

/**
 * Called by both transports with every error they are about to throw: the one place a
 * plan-limit refusal is handed to the app. When the handler shows the refusal, the error is
 * remembered so the caller's own error reporting can stay quiet instead of repeating it as a
 * generic toast.
 */
export function reportPlanLimit(error: object, request: PlanLimitRequest): boolean {
  const notice = planLimitFromError(error);
  if (!notice || !planLimitHandler) {
    return false;
  }

  let shown = false;
  try {
    shown = planLimitHandler(notice, request);
  } catch (handlerError) {
    console.error("[plan-limit] handler failed", handlerError);
  }
  if (shown) {
    markExplained(error);
  }
  return shown;
}

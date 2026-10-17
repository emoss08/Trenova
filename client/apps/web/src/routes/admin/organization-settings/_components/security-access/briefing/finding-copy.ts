import type { SecurityTabValue } from "@/hooks/use-organization-setting-state";
import { formatPeriod } from "@/lib/period";
import type { RichTag, RichTranslateFn } from "@trenova/shared/i18n/rich";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import type {
  SecurityBriefing,
  SecurityFinding,
  SecurityFindingSeverity,
} from "@trenova/shared/types/iam";

/**
 * Where a finding's next step goes: another tab of the security workspace, or a page of its own.
 * A page link names the permission it needs so the briefing can leave out a step the reader
 * could not take.
 */
export type FindingAction =
  | { kind: "tab"; tab: SecurityTabValue; label: string }
  | { kind: "page"; to: string; label: string; requires: "users" | "apiKeys" };

export type FindingCopy = {
  title: string;
  detail: string;
  action: FindingAction | null;
};

export const USERS_PAGE = "/admin/users";
export const API_KEYS_PAGE = "/admin/api-keys";

const SEVERITY_RANK: Record<SecurityFindingSeverity, number> = {
  critical: 0,
  warning: 1,
  notice: 2,
  pass: 3,
};

export type FindingGroups = {
  open: SecurityFinding[];
  passed: SecurityFinding[];
  critical: number;
  warning: number;
  notice: number;
};

/**
 * Splits the server's findings into what still wants doing and what already holds, keeping the
 * server's order within each (it ranks them) and counting each severity for the headline.
 */
export function groupFindings(findings: readonly SecurityFinding[]): FindingGroups {
  const open: SecurityFinding[] = [];
  const passed: SecurityFinding[] = [];
  let critical = 0;
  let warning = 0;
  let notice = 0;

  for (const finding of findings) {
    switch (finding.severity) {
      case "critical":
        critical += 1;
        open.push(finding);
        break;
      case "warning":
        warning += 1;
        open.push(finding);
        break;
      case "notice":
        notice += 1;
        open.push(finding);
        break;
      case "pass":
        passed.push(finding);
        break;
    }
  }

  open.sort((left, right) => SEVERITY_RANK[left.severity] - SEVERITY_RANK[right.severity]);

  return { open, passed, critical, warning, notice };
}

/**
 * Rounded down, so 199 of 200 reads 99% rather than a 100% that would hide the one person left.
 */
export function coveragePercent(enrolled: number, total: number): number {
  if (total <= 0) {
    return 0;
  }
  return Math.floor((Math.min(enrolled, total) * 100) / total);
}

/**
 * What a finding says to the person reading it. The server decides whether a check passes and
 * how severe it is; this only puts that into words and points at where it is fixed.
 */
export function describeFinding(
  t: TranslateFn,
  finding: SecurityFinding,
  briefing: SecurityBriefing,
): FindingCopy {
  const passed = finding.severity === "pass";
  const { count, total } = finding;

  switch (finding.code) {
    case "mfa_coverage":
      if (passed) {
        return {
          title: t("Every member has two-factor turned on"),
          detail: t(
            "{0, plural, one {The one member signs} other {All # members sign}} in with a second factor.",
            total,
          ),
          action: null,
        };
      }
      return {
        title: t(
          "{0, plural, one {# member hasn't turned on two-factor} other {# members haven't turned on two-factor}}",
          count,
        ),
        detail:
          briefing.sso?.enforced === true
            ? t(
                "Single sign-on is enforced, so they sign in through your identity provider and its own checks apply. Two-factor still protects them if password sign-in is ever turned back on.",
              )
            : t(
                "{0} of {1} members sign in with a password alone, so a stolen password is all it takes to get into their account. Each person turns two-factor on from their own security settings.",
                count,
                total,
              ),
        action: { kind: "page", to: USERS_PAGE, label: t("Open users"), requires: "users" },
      };

    case "mfa_policy":
      if (passed) {
        return {
          title: t(
            "{0, plural, one {# policy requires two-factor} other {# policies require two-factor}}",
            count,
          ),
          detail: t("Sensitive operations ask for a second factor before they go ahead."),
          action: null,
        };
      }
      return {
        title: t("No policy requires two-factor"),
        detail: t(
          "Turning on two-factor is left to each person. An access policy with the MFA required condition makes it a condition of the operations you choose.",
        ),
        action: { kind: "tab", tab: "policies", label: t("Open policies") },
      };

    case "sso":
      if (passed) {
        const name = briefing.sso?.enforcedProviderName ?? "";
        return {
          title: t("Single sign-on is enforced"),
          detail:
            name === ""
              ? t("Everyone signs in through your identity provider.")
              : t("Everyone signs in through {0}.", name),
          action: null,
        };
      }
      if ((briefing.sso?.enabledProviders ?? 0) === 0) {
        return {
          title: t("Single sign-on isn't set up"),
          detail: t(
            "Everyone keeps a separate password for Trenova. With Entra ID or Okta connected, your identity provider decides who gets in, and turning someone off there turns them off here.",
          ),
          action: { kind: "tab", tab: "sign-in", label: t("Add a provider") },
        };
      }
      return {
        title: t("Single sign-on is optional"),
        detail: t(
          "People can still sign in with a password, which goes around your identity provider's checks. Enforce single sign-on once everyone has signed in through it.",
        ),
        action: { kind: "tab", tab: "sign-in", label: t("Review providers") },
      };

    case "scim":
      if (passed) {
        return {
          title: t("Directory sync is on"),
          detail: t("Someone who leaves in your identity provider loses access here as well."),
          action: null,
        };
      }
      return {
        title: t("No directory sync"),
        detail: t(
          "Accounts aren't created or turned off when people join or leave in your identity provider. Connect a SCIM directory so a leaver loses access the same day.",
        ),
        action: { kind: "tab", tab: "provisioning", label: t("Connect a directory") },
      };

    case "dormant_accounts":
      if (passed) {
        return {
          title: t(
            "Every account has been used in the last {0, plural, one {# day} other {# days}}",
            briefing.dormantAfterDays,
          ),
          detail: t("No idle account is holding on to access."),
          action: null,
        };
      }
      return {
        title: t(
          "{0, plural, one {# account hasn't been used} other {# accounts haven't been used}} in {1, plural, one {# day} other {# days}}",
          count,
          briefing.dormantAfterDays,
        ),
        detail: t(
          "An unused account keeps every permission it was given. Turn off the ones nobody needs any more.",
        ),
        action: { kind: "page", to: USERS_PAGE, label: t("Review users"), requires: "users" },
      };

    case "locked_accounts":
      return {
        title: t("{0, plural, one {# account is locked} other {# accounts are locked}}", count),
        detail: t(
          "A locked account can't sign in until an administrator unlocks it. Check that each lock is expected.",
        ),
        action: { kind: "page", to: USERS_PAGE, label: t("Review users"), requires: "users" },
      };

    case "failed_sign_ins":
      if (passed) {
        return {
          title: t(
            "No failed sign-ins in the last {0, plural, one {# day} other {# days}}",
            briefing.signInWindowDays,
          ),
          detail: t("Every sign-in attempt in that time succeeded."),
          action: null,
        };
      }
      return {
        title: t(
          "{0, plural, one {# failed sign-in} other {# failed sign-ins}} in the last {1, plural, one {# day} other {# days}}",
          count,
          briefing.signInWindowDays,
        ),
        detail: t(
          "{0} of {1} sign-in attempts didn't succeed. Many from one address can be someone guessing passwords.",
          count,
          total,
        ),
        action: { kind: "tab", tab: "activity", label: t("Review activity") },
      };

    case "risk_denials":
      return {
        title: t(
          "{0, plural, one {# sign-in was blocked as risky} other {# sign-ins were blocked as risky}}",
          count,
        ),
        detail: t(
          "Risk checks refused these in the last {0, plural, one {# day} other {# days}}. Confirm they weren't your own people.",
          briefing.signInWindowDays,
        ),
        action: { kind: "tab", tab: "activity", label: t("Review activity") },
      };

    case "api_key_expiry":
      if (passed) {
        return {
          title: t("Every API key has an expiry date"),
          detail: t(
            "{0, plural, one {The active key stops} other {All # active keys stop}} working on a date you set.",
            total,
          ),
          action: null,
        };
      }
      return {
        title: t(
          "{0, plural, one {# API key never expires} other {# API keys never expire}}",
          count,
        ),
        detail: t(
          "A key without an expiry works until someone remembers to revoke it. Give each key an end date and rotate it before then.",
        ),
        action: {
          kind: "page",
          to: API_KEYS_PAGE,
          label: t("Review API keys"),
          requires: "apiKeys",
        },
      };

    case "api_key_unused":
      if (passed) {
        return {
          title: t("Every API key is in use"),
          detail: t(
            "Each active key has been used in the last {0, plural, one {# day} other {# days}}.",
            briefing.apiKeyUnusedAfterDays,
          ),
          action: null,
        };
      }
      return {
        title: t(
          "{0, plural, one {# API key hasn't been used} other {# API keys haven't been used}} in {1, plural, one {# day} other {# days}}",
          count,
          briefing.apiKeyUnusedAfterDays,
        ),
        detail: t("A key nobody uses still opens the API. Revoke the ones you no longer need."),
        action: {
          kind: "page",
          to: API_KEYS_PAGE,
          label: t("Review API keys"),
          requires: "apiKeys",
        },
      };

    case "api_key_expiring":
      return {
        title: t(
          "{0, plural, one {# API key expires} other {# API keys expire}} within {1, plural, one {# day} other {# days}}",
          count,
          briefing.apiKeyExpiringWithinDays,
        ),
        detail: t("Issue replacements before they expire so integrations keep running."),
        action: {
          kind: "page",
          to: API_KEYS_PAGE,
          label: t("Review API keys"),
          requires: "apiKeys",
        },
      };

    case "role_activation":
      if (passed) {
        return {
          title: t("Role activation is on"),
          detail: t("People work under only the roles they choose for each session."),
          action: null,
        };
      }
      return {
        title: t("Role activation is off"),
        detail: t(
          "Every role a person holds is in force at once. With role activation on, people work under only the roles they choose, which separation of duty needs.",
        ),
        action: { kind: "tab", tab: "policies", label: t("Open access settings") },
      };

    case "idle_timeout":
      if (passed) {
        return {
          title: t("Idle browsers are signed out"),
          detail: t(
            "A browser left untouched for {0} is signed out.",
            formatPeriod(t, briefing.session.idleTimeoutSeconds),
          ),
          action: null,
        };
      }
      return {
        title: t("Idle browsers stay signed in"),
        detail: t(
          "A browser left open stays signed in until its session ends. Idle sign-out is set by whoever runs this server, in security.session.idleTimeout.",
        ),
        action: null,
      };
  }
}

/** The address of a tab of the security workspace, so a phrase or a next step can link to it. */
export function securityTabHref(tab: SecurityTabValue): string {
  return `?tab=security&securityTab=${tab}`;
}

/** Where a finding's next step leads, as a link. */
export function findingHref(action: FindingAction): string {
  return action.kind === "tab" ? securityTabHref(action.tab) : action.to;
}

/** How many findings the brief puts into words; the rest are left to the list under it. */
export const BRIEF_SENTENCE_LIMIT = 4;

/**
 * One finding as a sentence of the brief, with the phrase that names it wrapped in `phrase` so
 * the caller can make it a link (or plain emphasis when the reader cannot follow it). Each is
 * one whole message, so a translator sees the sentence and moves the phrase with the words.
 * A passed check, and a finding whose only remedy is outside the product, read as plain text.
 */
export function briefSentence(
  rt: RichTranslateFn,
  t: TranslateFn,
  finding: SecurityFinding,
  briefing: SecurityBriefing,
  phrase: RichTag,
): ReactNode {
  const tags = { a: phrase };
  const { count } = finding;

  switch (finding.code) {
    case "mfa_coverage":
      return briefing.sso?.enforced === true
        ? rt(
            "<a>{0, plural, one {# member hasn't} other {# members haven't}} turned on two-factor</a>, though single sign-on covers them for now.",
            tags,
            count,
          )
        : rt(
            "<a>{0, plural, one {# member hasn't} other {# members haven't}} turned on two-factor</a>, so a stolen password is all it takes to get in.",
            tags,
            count,
          );
    case "mfa_policy":
      return rt("No access policy <a>requires two-factor</a> yet.", tags);
    case "sso":
      return (briefing.sso?.enabledProviders ?? 0) === 0
        ? rt("Single sign-on <a>isn't set up</a>, so everyone keeps a separate password.", tags)
        : rt(
            "Single sign-on is on but <a>not enforced</a>, so people can still sign in with a password.",
            tags,
          );
    case "scim":
      return rt(
        "Accounts <a>don't sync from your directory</a>, so a leaver keeps access until someone turns it off.",
        tags,
      );
    case "dormant_accounts":
      return rt(
        "<a>{0, plural, one {# account hasn't} other {# accounts haven't}} been used in {1, plural, one {# day} other {# days}}</a> but still {0, plural, one {holds its} other {hold their}} access.",
        tags,
        count,
        briefing.dormantAfterDays,
      );
    case "locked_accounts":
      return rt(
        "<a>{0, plural, one {# account is} other {# accounts are}} locked</a> and waiting on an administrator.",
        tags,
        count,
      );
    case "failed_sign_ins":
      return rt(
        "<a>{0, plural, one {# sign-in} other {# sign-ins}} failed</a> in the last {1, plural, one {# day} other {# days}}.",
        tags,
        count,
        briefing.signInWindowDays,
      );
    case "risk_denials":
      return rt(
        "Risk checks <a>blocked {0, plural, one {# sign-in} other {# sign-ins}}</a> in the last {1, plural, one {# day} other {# days}}.",
        tags,
        count,
        briefing.signInWindowDays,
      );
    case "api_key_expiry":
      return rt(
        "<a>{0, plural, one {# API key never expires} other {# API keys never expire}}</a>.",
        tags,
        count,
      );
    case "api_key_unused":
      return rt(
        "<a>{0, plural, one {# API key hasn't} other {# API keys haven't}} been used in {1, plural, one {# day} other {# days}}</a>.",
        tags,
        count,
        briefing.apiKeyUnusedAfterDays,
      );
    case "api_key_expiring":
      return rt(
        "<a>{0, plural, one {# API key expires} other {# API keys expire}}</a> within {1, plural, one {# day} other {# days}}.",
        tags,
        count,
        briefing.apiKeyExpiringWithinDays,
      );
    case "role_activation":
      return rt(
        "Role activation <a>is off</a>, so every role a person holds is in force at once.",
        tags,
      );
    case "idle_timeout":
      return t("Idle browsers stay signed in until their session ends.");
  }
}

/** The sentence that closes the brief: what holds, and what the brief left to the list. */
export function briefClosing(t: TranslateFn, groups: FindingGroups): string {
  if (groups.open.length === 0) {
    return t("Everything the briefing checks is in order.");
  }
  const unsaid = Math.max(0, groups.open.length - BRIEF_SENTENCE_LIMIT);
  if (unsaid > 0) {
    return t(
      "{0, plural, one {# more recommendation is} other {# more recommendations are}} listed below.",
      unsaid,
    );
  }
  if (groups.passed.length > 0) {
    return t(
      "{0, plural, one {The other check passes.} other {The other # checks pass.}}",
      groups.passed.length,
    );
  }
  return "";
}

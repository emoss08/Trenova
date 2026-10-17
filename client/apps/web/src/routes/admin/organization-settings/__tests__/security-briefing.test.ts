import { formatPeriod } from "@/lib/period";
import { translate } from "@trenova/shared/i18n/runtime";
import {
  securityBriefingSchema,
  type SecurityBriefing,
  type SecurityFinding,
} from "@trenova/shared/types/iam";
import { describe, expect, it } from "vitest";
import {
  API_KEYS_PAGE,
  BRIEF_SENTENCE_LIMIT,
  briefClosing,
  coveragePercent,
  describeFinding,
  findingHref,
  groupFindings,
  securityTabHref,
  USERS_PAGE,
} from "../_components/security-access/briefing/finding-copy";

/** The briefing as `GET /organizations/:id/iam/security-briefing` returns it to a full reader. */
const wire = {
  generatedAt: 1_760_000_000,
  signInWindowDays: 7,
  dormantAfterDays: 90,
  apiKeyUnusedAfterDays: 90,
  apiKeyExpiringWithinDays: 14,
  findings: [
    { code: "mfa_coverage", severity: "critical", count: 30, total: 40 },
    { code: "sso", severity: "notice", count: 1, total: 0 },
    { code: "idle_timeout", severity: "pass", count: 0, total: 0 },
  ],
  session: { maxAgeSeconds: 43_200, idleTimeoutSeconds: 1_800, idleWarningSeconds: 120 },
  members: { active: 40, locked: 1, neverSignedIn: 3, dormant: 5, mustChangePassword: 0 },
  mfa: {
    enrolled: 10,
    total: 40,
    unenrolled: [
      {
        userId: "usr_01",
        name: "Dana Ruiz",
        emailAddress: "dana@example.com",
        lastLoginAt: 1_759_900_000,
      },
      { userId: "usr_02", name: "Sam Lee", emailAddress: "sam@example.com", lastLoginAt: null },
    ],
  },
  signIns: { succeeded: 300, failed: 9, denied: 2, throttled: 1, riskDenied: 0 },
  apiKeys: { active: 4, withoutExpiry: 2, unused: 1, expiringSoon: 0 },
  sso: {
    enabledProviders: 1,
    enforced: false,
    enforcedProviderName: "",
    trustsFederatedMfa: false,
  },
  provisioning: { activeDirectories: 0 },
  policies: { enabled: 3, requiringMfa: 0, roleActivationRequired: false },
};

function parse(input: unknown): SecurityBriefing {
  return securityBriefingSchema.parse(input);
}

function finding(
  code: SecurityFinding["code"],
  severity: SecurityFinding["severity"],
  count = 0,
  total = 0,
): SecurityFinding {
  return { code, severity, count, total };
}

describe("securityBriefingSchema", () => {
  it("reads every section of a full briefing", () => {
    const briefing = parse(wire);

    expect(briefing.findings).toHaveLength(3);
    expect(briefing.mfa?.unenrolled?.[1]).toEqual({
      userId: "usr_02",
      name: "Sam Lee",
      emailAddress: "sam@example.com",
      lastLoginAt: null,
    });
    expect(briefing.signIns?.riskDenied).toBe(0);
  });

  it("keeps a section the reader may not see as null rather than as zeros", () => {
    const briefing = parse({
      ...wire,
      members: null,
      mfa: null,
      signIns: null,
      apiKeys: null,
      sso: null,
      provisioning: null,
      policies: null,
    });

    expect(briefing.members).toBeNull();
    expect(briefing.mfa).toBeNull();
    expect(briefing.apiKeys).toBeNull();
    expect(briefing.policies).toBeNull();
  });

  it("tells a hidden list of names apart from an empty one", () => {
    const hidden = parse({ ...wire, mfa: { enrolled: 10, total: 40, unenrolled: null } });
    const empty = parse({ ...wire, mfa: { enrolled: 40, total: 40, unenrolled: [] } });

    expect(hidden.mfa?.unenrolled).toBeNull();
    expect(empty.mfa?.unenrolled).toEqual([]);
  });

  it("reads a risk count the reader may not see as null, whether sent null or left out", () => {
    const sentNull = parse({ ...wire, signIns: { ...wire.signIns, riskDenied: null } });
    const { riskDenied: _omitted, ...withoutRisk } = wire.signIns;
    const leftOut = parse({ ...wire, signIns: withoutRisk });

    expect(sentNull.signIns?.riskDenied).toBeNull();
    expect(leftOut.signIns?.riskDenied).toBeNull();
  });

  it("drops a finding a newer server knows and this client does not", () => {
    const briefing = parse({
      ...wire,
      findings: [
        { code: "passkeys", severity: "warning", count: 2, total: 5 },
        { code: "locked_accounts", severity: "notice", count: 1, total: 0 },
      ],
    });

    expect(briefing.findings).toEqual([finding("locked_accounts", "notice", 1, 0)]);
  });

  it("refuses a severity outside the contract", () => {
    expect(() =>
      parse({ ...wire, findings: [{ code: "sso", severity: "urgent", count: 0, total: 0 }] }),
    ).toThrow();
  });
});

describe("groupFindings", () => {
  it("separates passed checks and counts each open severity", () => {
    const groups = groupFindings([
      finding("sso", "notice"),
      finding("mfa_coverage", "critical", 30, 40),
      finding("idle_timeout", "pass"),
      finding("api_key_expiry", "warning", 2, 4),
      finding("scim", "notice"),
    ]);

    expect(groups.open.map((item) => item.code)).toEqual([
      "mfa_coverage",
      "api_key_expiry",
      "sso",
      "scim",
    ]);
    expect(groups.passed.map((item) => item.code)).toEqual(["idle_timeout"]);
    expect({ critical: groups.critical, warning: groups.warning, notice: groups.notice }).toEqual({
      critical: 1,
      warning: 1,
      notice: 2,
    });
  });
});

describe("briefClosing", () => {
  it("says everything is in order when no check is open", () => {
    expect(briefClosing(translate, groupFindings([finding("sso", "pass")]))).toBe(
      "Everything the briefing checks is in order.",
    );
  });

  it("names how many checks pass once the open ones are all said", () => {
    expect(
      briefClosing(
        translate,
        groupFindings([finding("sso", "notice"), finding("idle_timeout", "pass"), finding("scim", "pass")]),
      ),
    ).toBe("The other 2 checks pass.");
  });

  it("points to the list for recommendations the brief left unsaid", () => {
    const open = [
      finding("mfa_coverage", "critical", 30, 40),
      finding("dormant_accounts", "warning", 5, 40),
      finding("api_key_expiry", "warning", 2, 4),
      finding("sso", "notice"),
      finding("scim", "notice"),
      finding("role_activation", "notice"),
    ];

    expect(open.length - BRIEF_SENTENCE_LIMIT).toBe(2);
    expect(briefClosing(translate, groupFindings(open))).toBe(
      "2 more recommendations are listed below.",
    );
  });

  it("says nothing more when every open check is said and none passed", () => {
    expect(briefClosing(translate, groupFindings([finding("sso", "notice")]))).toBe("");
  });
});

describe("findingHref", () => {
  it("links a tab of the security workspace through the page's own search parameters", () => {
    expect(securityTabHref("policies")).toBe("?tab=security&securityTab=policies");
    expect(findingHref({ kind: "tab", tab: "sign-in", label: "Add a provider" })).toBe(
      "?tab=security&securityTab=sign-in",
    );
  });

  it("links a page by its path", () => {
    expect(
      findingHref({ kind: "page", to: USERS_PAGE, label: "Open users", requires: "users" }),
    ).toBe("/admin/users");
  });
});

describe("describeFinding", () => {
  const briefing = parse(wire);

  it("counts members without two-factor against the whole membership", () => {
    const copy = describeFinding(translate, finding("mfa_coverage", "critical", 30, 40), briefing);

    expect(copy.title).toBe("30 members haven't turned on two-factor");
    expect(copy.detail).toContain("30 of 40 members sign in with a password alone");
    expect(copy.action).toEqual({
      kind: "page",
      to: USERS_PAGE,
      label: "Open users",
      requires: "users",
    });
  });

  it("explains that the identity provider applies its own checks when single sign-on is enforced", () => {
    const enforced = parse({
      ...wire,
      sso: { ...wire.sso, enforced: true, enforcedProviderName: "Entra ID" },
    });
    const copy = describeFinding(translate, finding("mfa_coverage", "notice", 30, 40), enforced);

    expect(copy.detail).toContain("Single sign-on is enforced");
  });

  it("asks for a provider when single sign-on has none", () => {
    const none = parse({ ...wire, sso: { ...wire.sso, enabledProviders: 0 } });
    const copy = describeFinding(translate, finding("sso", "notice"), none);

    expect(copy.title).toBe("Single sign-on isn't set up");
    expect(copy.action).toEqual({ kind: "tab", tab: "sign-in", label: "Add a provider" });
  });

  it("calls single sign-on optional when a provider is on but not enforced", () => {
    const copy = describeFinding(translate, finding("sso", "notice", 1), briefing);

    expect(copy.title).toBe("Single sign-on is optional");
    expect(copy.action).toEqual({ kind: "tab", tab: "sign-in", label: "Review providers" });
  });

  it("names the enforced provider once single sign-on passes", () => {
    const enforced = parse({
      ...wire,
      sso: { ...wire.sso, enforced: true, enforcedProviderName: "Okta" },
    });
    const copy = describeFinding(translate, finding("sso", "pass", 1), enforced);

    expect(copy.detail).toBe("Everyone signs in through Okta.");
    expect(copy.action).toBeNull();
  });

  it("uses the server's windows in the sentence, not numbers of its own", () => {
    const copy = describeFinding(translate, finding("failed_sign_ins", "warning", 12, 312), briefing);

    expect(copy.title).toBe("12 failed sign-ins in the last 7 days");
    expect(copy.detail).toBe(
      "12 of 312 sign-in attempts didn't succeed. Many from one address can be someone guessing passwords.",
    );
    expect(copy.action).toEqual({ kind: "tab", tab: "activity", label: "Review activity" });
  });

  it("sends API key findings to the API keys page", () => {
    const copy = describeFinding(translate, finding("api_key_expiry", "warning", 2, 4), briefing);

    expect(copy.title).toBe("2 API keys never expire");
    expect(copy.action).toEqual({
      kind: "page",
      to: API_KEYS_PAGE,
      label: "Review API keys",
      requires: "apiKeys",
    });
  });

  it("offers no step for the idle timeout, which only the server's operator can change", () => {
    const off = parse({ ...wire, session: { ...wire.session, idleTimeoutSeconds: 0 } });

    expect(describeFinding(translate, finding("idle_timeout", "warning"), off).action).toBeNull();
    expect(describeFinding(translate, finding("idle_timeout", "pass"), briefing).detail).toBe(
      "A browser left untouched for 30 minutes is signed out.",
    );
  });
});

describe("coveragePercent", () => {
  it("rounds down so one person left is never shown as full coverage", () => {
    expect(coveragePercent(199, 200)).toBe(99);
    expect(coveragePercent(200, 200)).toBe(100);
  });

  it("reads an empty membership as no coverage", () => {
    expect(coveragePercent(0, 0)).toBe(0);
  });

  it("never passes 100 when the counts disagree", () => {
    expect(coveragePercent(41, 40)).toBe(100);
  });
});

describe("formatPeriod", () => {
  it("reads a period back in the largest unit that divides it", () => {
    expect(formatPeriod(translate, 43_200)).toBe("12 hours");
    expect(formatPeriod(translate, 86_400)).toBe("1 day");
    expect(formatPeriod(translate, 1_800)).toBe("30 minutes");
    expect(formatPeriod(translate, 5_400)).toBe("90 minutes");
    expect(formatPeriod(translate, 90)).toBe("90 seconds");
  });

  it("reads nothing, or less than nothing, as zero seconds", () => {
    expect(formatPeriod(translate, 0)).toBe("0 seconds");
    expect(formatPeriod(translate, -5)).toBe("0 seconds");
  });
});

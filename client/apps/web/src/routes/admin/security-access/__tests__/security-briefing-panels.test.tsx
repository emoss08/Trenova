import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { SecurityBriefing } from "@trenova/shared/types/iam";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/user-avatars", () => ({ loadUserAvatar: async () => null }));
import { UnenrolledMembersPanel } from "../_components/security-access/briefing/briefing-aside";
import { PostureRing } from "../_components/security-access/briefing/posture-ring";
import {
  BriefingBrief,
  type ReachableAction,
} from "../_components/security-access/briefing/briefing-brief";
import { BriefingFindings } from "../_components/security-access/briefing/briefing-findings";
import {
  describeFinding,
  findingHref,
  groupFindings,
} from "../_components/security-access/briefing/finding-copy";

const briefing: SecurityBriefing = {
  generatedAt: 1_760_000_000,
  signInWindowDays: 7,
  dormantAfterDays: 90,
  apiKeyUnusedAfterDays: 90,
  apiKeyExpiringWithinDays: 14,
  findings: [
    { code: "mfa_coverage", severity: "warning", count: 3, total: 40 },
    { code: "mfa_policy", severity: "notice", count: 0, total: 0 },
    { code: "idle_timeout", severity: "pass", count: 0, total: 0 },
    { code: "role_activation", severity: "pass", count: 0, total: 0 },
  ],
  session: { maxAgeSeconds: 43_200, idleTimeoutSeconds: 1_800, idleWarningSeconds: 120 },
  members: null,
  mfa: { enrolled: 37, total: 40, unenrolled: null },
  signIns: null,
  apiKeys: null,
  sso: null,
  provisioning: null,
  policies: { enabled: 2, requiringMfa: 0, roleActivationRequired: true },
};

/** Everything reachable, as for an administrator who holds every read. */
const reachEverything: ReachableAction = (finding) => {
  const action = describeFinding(translate, finding, briefing).action;
  return action ? findingHref(action) : null;
};

const reachNothing: ReachableAction = () => null;

function renderFindings(hrefFor: ReachableAction) {
  render(
    <TestRouter>
      <BriefingFindings
        briefing={briefing}
        groups={groupFindings(briefing.findings)}
        hrefFor={hrefFor}
      />
    </TestRouter>,
  );
}

function renderBrief(hrefFor: ReachableAction) {
  useAuthStore.setState({ user: { name: "Dana Ruiz" } as unknown as User });
  render(
    <TestRouter>
      <BriefingBrief
        briefing={briefing}
        groups={groupFindings(briefing.findings)}
        hrefFor={hrefFor}
        refreshing={false}
        onRefresh={vi.fn()}
      />
    </TestRouter>,
  );
}

describe("BriefingBrief", () => {
  afterEach(() => {
    useAuthStore.setState({ user: null });
  });

  it("greets the reader and says each open finding as a sentence", () => {
    renderBrief(reachEverything);

    const brief = screen.getByRole("region", { name: "Security brief" });
    expect(brief).toHaveTextContent(
      "Hello Dana. 3 members haven't turned on two-factor, so a stolen password is all it takes to get in. No access policy requires two-factor yet. The other 2 checks pass.",
    );
  });

  it("links the phrase that names a finding to where it is settled", () => {
    renderBrief(reachEverything);

    expect(
      screen.getByRole("link", { name: "3 members haven't turned on two-factor" }),
    ).toHaveAttribute("href", "/admin/users");
    expect(screen.getByRole("link", { name: "requires two-factor" })).toHaveAttribute(
      "href",
      "/admin/security-access?tab=policies",
    );
  });

  it("keeps the sentence but drops the links a reader could not follow", () => {
    renderBrief(reachNothing);

    expect(screen.getByRole("region", { name: "Security brief" })).toHaveTextContent(
      "3 members haven't turned on two-factor",
    );
    expect(screen.queryByRole("link")).toBeNull();
  });
});

describe("BriefingFindings", () => {
  it("makes each open finding a row that opens where it is settled", () => {
    renderFindings(reachEverything);

    expect(
      screen.getByRole("link", { name: "3 members haven't turned on two-factor: Open users" }),
    ).toHaveAttribute("href", "/admin/users");
    expect(
      screen.getByRole("link", { name: "No policy requires two-factor: Open policies" }),
    ).toHaveAttribute("href", "/admin/security-access?tab=policies");
  });

  it("draws a finding the reader cannot act on as plain text", () => {
    renderFindings(reachNothing);

    expect(screen.getByText("3 members haven't turned on two-factor")).toBeInTheDocument();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("folds passed checks away behind their count", () => {
    renderFindings(reachEverything);

    expect(screen.getByRole("button", { name: "2 checks passed" })).toBeInTheDocument();
    expect(screen.queryByText("Role activation is on")).toBeNull();
  });
});

describe("UnenrolledMembersPanel", () => {
  it("draws a member's photo from the links the briefing carries", () => {
    const { container } = render(
      <QueryClientProvider client={new QueryClient()}>
        <TestRouter>
          <UnenrolledMembersPanel
            canOpenUsers
            mfa={{
              enrolled: 1,
              total: 2,
              unenrolled: [
                {
                  userId: "usr_02",
                  name: "Lee Park",
                  emailAddress: "lee@example.com",
                  lastLoginAt: null,
                  photo: {
                    url: "https://files.example/usr_02/photo.jpg?sig=a",
                    thumbnailUrl: "https://files.example/usr_02/thumb.jpg?sig=a",
                  },
                },
              ],
            }}
          />
        </TestRouter>
      </QueryClientProvider>,
    );

    expect(container.querySelector("img")?.getAttribute("src")).toBe(
      "https://files.example/usr_02/thumb.jpg?sig=a",
    );
  });

  it("says how many more there are than the server listed", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <TestRouter>
          <UnenrolledMembersPanel
            canOpenUsers
            mfa={{
              enrolled: 10,
              total: 40,
              unenrolled: [
                {
                  userId: "usr_01",
                  name: "Dana Ruiz",
                  emailAddress: "dana@example.com",
                  lastLoginAt: null,
                  photo: null,
                },
              ],
            }}
          />
        </TestRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Dana Ruiz")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Dana Ruiz" })).toBeInTheDocument();
    expect(screen.getByTitle("Never signed in")).toHaveTextContent("Never");
    expect(screen.getByText("and 29 more")).toBeInTheDocument();
  });

  it("keeps names back from a reader who cannot read users", () => {
    render(
      <TestRouter>
        <UnenrolledMembersPanel
          canOpenUsers={false}
          mfa={{ enrolled: 37, total: 40, unenrolled: null }}
        />
      </TestRouter>,
    );

    expect(
      screen.getByText("Their names are shown to people who can read users."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Users" })).toBeNull();
  });

  it("draws nothing once every member has two-factor", () => {
    const { container } = render(
      <TestRouter>
        <UnenrolledMembersPanel canOpenUsers mfa={{ enrolled: 40, total: 40, unenrolled: [] }} />
      </TestRouter>,
    );

    expect(container).toBeEmptyDOMElement();
  });
});

describe("PostureRing", () => {
  it("counts every check the briefing ran and how many passed", () => {
    render(<PostureRing groups={groupFindings(briefing.findings)} />);

    expect(screen.getByRole("img", { name: "2 of 4 checks pass" })).toBeInTheDocument();
    const legend = screen.getByRole("list", { name: "Check outcomes" });
    expect(legend).toHaveTextContent("1 need attention");
    expect(legend).toHaveTextContent("1 recommended");
    expect(legend).toHaveTextContent("2 passed");
    expect(legend).not.toHaveTextContent("critical");
  });
});

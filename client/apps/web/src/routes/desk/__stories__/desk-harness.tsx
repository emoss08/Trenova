/* eslint-disable */
// The Desk, served a morning's worth of fixture data.
//
// Every request the Desk makes is answered here: REST by path, GraphQL by
// operation name. Nothing reaches a server, so a story renders the same way
// every time and a screenshot of it is a screenshot of the design.
import { DeskPage } from "@/routes/desk/page";
import { DeskConversationPage } from "@/routes/desk/conversation-page";
import { DeskDecisionsPage } from "@/routes/desk/decisions-page";
import { DeskHomePage } from "@/routes/desk/home-page";
import { DeskWatchtowerPage } from "@/routes/desk/watchtower-page";
import { useDeskStore } from "@/stores/desk-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { LazyMotion, domMax } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { RouterProvider, createMemoryRouter } from "react-router";
import {
  AGENTS,
  ARTIFACTS,
  ATTENTION,
  BRIEFING,
  LIVE_TURNS,
  MESSAGES,
  PENDING_SUMMARY,
  PLANS,
  PROPOSALS,
  PROPOSAL_PREVIEW,
  PROVIDERS,
  THREADS,
  THREAD_ID,
  USER,
  WATCHTOWER_COUNTS,
  WATCHTOWER_ITEMS,
} from "./desk-fixtures";

type Json = Record<string, unknown> | unknown[] | null;

const page = (edges: unknown[], total = edges.length) => ({
  edges: edges.map((node) => ({ cursor: "c", node })),
  totalCount: total,
  pageInfo: { hasNextPage: false, endCursor: null, hasPreviousPage: false, startCursor: null },
});

function graphql(operationName: string, variables: Record<string, unknown>): Json {
  switch (operationName) {
    case "MyAgents":
      return { myAgents: page(AGENTS) };
    case "AgentChoices":
      return { agentDefinitions: page(AGENTS.map(({ delegates, ...agent }) => agent)) };
    case "AgentDefinitionCards":
      return { agentDefinitions: page(AGENTS) };
    case "AttentionSummary":
      return { attentionSummary: ATTENTION };
    case "TodaysBriefing":
      return { todaysBriefing: BRIEFING };
    case "WatchtowerCounts":
      return { watchtowerCounts: WATCHTOWER_COUNTS };
    case "WatchtowerFeed":
      return { watchtowerItems: { ...page(WATCHTOWER_ITEMS), seenAt: WATCHTOWER_COUNTS.seenAt } };
    case "PendingDecisionSummary":
      return { pendingDecisionSummary: PENDING_SUMMARY };
    case "PendingDecisions":
      return { pendingDecisions: page([]) };
    case "MyProposalPreview":
      return { myProposalPreview: PROPOSAL_PREVIEW };
    case "MyAIFeedback":
      return { myAIFeedback: null };
    default:
      console.warn("[desk story] unmocked GraphQL operation", operationName, variables);
      return {};
  }
}

function rest(method: string, path: string): { status: number; body: Json } {
  const ok = (body: Json) => ({ status: 200, body });
  if (path.endsWith("/assistant/threads/") || /\/assistant\/threads\/\?/.test(path)) {
    return ok({ items: THREADS, total: THREADS.length });
  }
  const thread = path.match(/\/assistant\/threads\/([^/]+)\/$/);
  if (thread) {
    const found = THREADS.find((t) => t.id === decodeURIComponent(thread[1]));
    return found ? ok(found) : { status: 404, body: { error: "not found" } };
  }
  if (/\/assistant\/threads\/[^/]+\/artifacts\//.test(path)) {
    return ok({ results: path.includes(THREAD_ID) ? ARTIFACTS : [] });
  }
  if (/\/assistant\/threads\/[^/]+\/messages\//.test(path)) {
    return ok({ results: path.includes(THREAD_ID) ? MESSAGES : [], hasMore: false, total: MESSAGES.length, limit: 50 });
  }
  if (/\/assistant\/threads\/[^/]+\/proposals\//.test(path)) {
    return ok({ results: path.includes(THREAD_ID) ? servedProposals : [] });
  }
  if (/\/assistant\/threads\/[^/]+\/plans\//.test(path)) {
    return ok({ results: path.includes(THREAD_ID) ? PLANS : [] });
  }
  if (/\/assistant\/threads\/[^/]+\/turns\/active\//.test(path)) {
    return ok({ items: [] });
  }
  if (path.endsWith("/assistant/turns/active/")) {
    return ok({ items: LIVE_TURNS });
  }
  if (path.endsWith("/assistant/providers/")) {
    return ok({ results: PROVIDERS });
  }
  if (path.includes("/auth/csrf")) {
    return ok({ csrfToken: "story" });
  }
  console.warn("[desk story] unmocked request", method, path);
  return { status: 404, body: { error: "not mocked" } };
}

/** Which of the fixture's proposals a story serves. */
let servedProposals = PROPOSALS;

let installed = false;
function installFetchStub() {
  if (installed) return;
  installed = true;
  const realFetch = window.fetch.bind(window);
  window.fetch = async (input, init) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const method = init?.method ?? "GET";
    if (url.includes("/graphql")) {
      let body: { operationName?: string; variables?: Record<string, unknown> } = {};
      try {
        body = JSON.parse(String(init?.body ?? "{}"));
      } catch {
        // an empty body is a query with nothing to say
      }
      const data = graphql(body.operationName ?? "", body.variables ?? {});
      return new Response(JSON.stringify({ data }), {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    }
    if (url.includes("/api/v1/") || url.includes("localhost:8080")) {
      const { status, body } = rest(method, url);
      return new Response(JSON.stringify(body), {
        status,
        headers: { "content-type": "application/json" },
      });
    }
    return realFetch(input, init);
  };
}

const ALLOW_ALL = {
  manifest: { permissions: {}, version: 1, routes: [] } as never,
  hasPermission: () => true,
  hasAnyPermission: () => true,
  hasAllPermissions: () => true,
  canAccessRoute: () => true,
};

export type DeskStoryOptions = {
  /** Where the Desk opens. */
  path?: string;
  /** Whether the rail stands open. */
  rail?: "open" | "closed";
  /** Whether the workspace stands open beside a conversation. */
  pane?: "open" | "closed";
  /** Which workspace tab is open. */
  tab?: "artifacts" | "decisions" | "activity";
  /** Whether a proposal is still waiting on the person, which puts the approval box in the composer's place. */
  decisions?: "waiting" | "settled";
};

/** The Desk at a path, with every store and request it reads answered. */
export function DeskStory({
  path = "/desk",
  rail = "open",
  pane = "open",
  tab = "artifacts",
  decisions = "waiting",
}: DeskStoryOptions) {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    servedProposals =
      decisions === "waiting"
        ? PROPOSALS
        : PROPOSALS.filter((proposal) => proposal.status !== "Pending");
    installFetchStub();
    useAuthStore.setState({ user: USER as never, isAuthenticated: true } as never);
    usePermissionStore.setState(ALLOW_ALL as never);
    useDeskStore.setState({ rail, pane, workspaceTab: tab } as never);
    setReady(true);
  }, [rail, pane, tab, decisions]);

  const router = useMemo(
    () =>
      createMemoryRouter(
        [
          {
            path: "/desk",
            Component: DeskPage,
            children: [
              { index: true, Component: DeskHomePage },
              { path: "t/:threadId", Component: DeskConversationPage },
              { path: "decisions", Component: DeskDecisionsPage },
              { path: "watchtower", Component: DeskWatchtowerPage },
            ],
          },
          { path: "*", element: <div className="p-6 text-sm">Left the Desk: {path}</div> },
        ],
        { initialEntries: [path] },
      ),
    [path],
  );

  if (!ready) return null;

  return (
    <LazyMotion features={domMax} strict>
      <div className="fixed inset-0">
        <RouterProvider router={router} />
      </div>
    </LazyMotion>
  );
}

export { THREAD_ID };

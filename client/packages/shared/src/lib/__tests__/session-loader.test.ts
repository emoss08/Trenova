import { ApiRequestError } from "@trenova/shared/lib/api";
import {
  pathScopedLoader,
  requireSession,
  revalidateOnPathChange,
  withPathScopedRevalidation,
} from "@trenova/shared/lib/session-loader";
import { useAuthStore, type SessionCheck } from "@trenova/shared/stores/auth-store";
import type { RouteObject, ShouldRevalidateFunctionArgs } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

const LOGIN_PATH = "/somewhere/login";

const rateLimited = new ApiRequestError(
  429,
  {
    type: "https://api.trenova.app/problems/rate-limit-exceeded",
    title: "Rate Limit Exceeded",
    status: 429,
    detail: "Too many requests",
  },
  7,
);

const originalCheckSession = useAuthStore.getState().checkSession;

// Sets the state first and the check last: the store replaces its state object on every
// write, so the check has to be the last thing written.
function arrange(outcome: SessionCheck, isAuthenticated: boolean) {
  useAuthStore.setState({ isAuthenticated });
  useAuthStore.setState({ checkSession: async () => outcome });
}

async function settle(promise: Promise<Response | null>) {
  return promise.then(
    (value) => ({ value, thrown: undefined }),
    (thrown: unknown) => ({ value: undefined, thrown }),
  );
}

describe("requireSession", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      checkSession: originalCheckSession,
    });
  });

  it("lets a signed-in session through", async () => {
    arrange({ status: "signed_in" }, true);

    await expect(requireSession(LOGIN_PATH)).resolves.toBeNull();
  });

  it("sends a signed-out session to the app's login path", async () => {
    arrange({ status: "signed_out" }, false);

    const result = await requireSession(LOGIN_PATH);

    expect(result).toBeInstanceOf(Response);
    expect(result?.status).toBe(302);
    expect(result?.headers.get("Location")).toBe(LOGIN_PATH);
  });

  // Mid-session navigation: the store already holds a working session, so a check that
  // could not reach a verdict leaves it standing and the page's own queries retry.
  it("carries on mid-session when the check could not tell", async () => {
    arrange({ status: "unreachable", error: rateLimited }, true);

    await expect(requireSession(LOGIN_PATH)).resolves.toBeNull();
  });

  it("carries on mid-session through a network failure", async () => {
    arrange({ status: "unreachable", error: new TypeError("Failed to fetch") }, true);

    await expect(requireSession(LOGIN_PATH)).resolves.toBeNull();
  });

  // A cold load has nothing that proves a session, and nothing that disproves one either.
  // The route's error boundary presents the original failure with its retry; the login
  // form would be a lie.
  it("throws the original error on a cold load when the check could not tell", async () => {
    arrange({ status: "unreachable", error: rateLimited }, false);

    const { value, thrown } = await settle(requireSession(LOGIN_PATH));

    expect(value).toBeUndefined();
    expect(thrown).toBe(rateLimited);
  });

  it("throws a network failure on a cold load rather than redirecting", async () => {
    const failure = new TypeError("Failed to fetch");
    arrange({ status: "unreachable", error: failure }, false);

    const { value, thrown } = await settle(requireSession(LOGIN_PATH));

    expect(value).toBeUndefined();
    expect(thrown).toBe(failure);
  });
});

function revalidation(
  from: string,
  to: string,
  extra: Partial<ShouldRevalidateFunctionArgs> = {},
): ShouldRevalidateFunctionArgs {
  return {
    currentUrl: new URL(from, "http://localhost"),
    nextUrl: new URL(to, "http://localhost"),
    currentParams: {},
    nextParams: {},
    defaultShouldRevalidate: true,
    ...extra,
  };
}

describe("revalidateOnPathChange", () => {
  it("skips a navigation that only changes the search", () => {
    expect(revalidateOnPathChange(revalidation("/triage?message=a", "/triage?message=b"))).toBe(
      false,
    );
  });

  it("skips a navigation that adds or drops the search", () => {
    expect(revalidateOnPathChange(revalidation("/triage", "/triage?message=a"))).toBe(false);
    expect(revalidateOnPathChange(revalidation("/triage?message=a", "/triage"))).toBe(false);
  });

  it("skips a navigation that only changes the hash", () => {
    expect(revalidateOnPathChange(revalidation("/triage#a", "/triage#b"))).toBe(false);
  });

  it("defers to the router when the path changes", () => {
    expect(revalidateOnPathChange(revalidation("/triage?message=a", "/intake?message=a"))).toBe(
      true,
    );
    expect(
      revalidateOnPathChange(
        revalidation("/triage", "/intake", { defaultShouldRevalidate: false }),
      ),
    ).toBe(false);
  });

  // A navigation to the URL already shown (keys pressed faster than navigations commit) is
  // indistinguishable from revalidate() here; the session and permission checks answer the
  // same either way, and a 401 on any request still ends the session.
  it("skips a navigation to the URL already shown", () => {
    expect(revalidateOnPathChange(revalidation("/triage?message=a", "/triage?message=a"))).toBe(
      false,
    );
  });

  it("defers to the router after a form submission on the same path", () => {
    expect(
      revalidateOnPathChange(
        revalidation("/triage?message=a", "/triage?message=b", { formMethod: "POST" }),
      ),
    ).toBe(true);
    expect(
      revalidateOnPathChange(
        revalidation("/triage?message=a", "/triage?message=b", { actionResult: { ok: true } }),
      ),
    ).toBe(true);
  });
});

describe("withPathScopedRevalidation", () => {
  const guard = pathScopedLoader(async () => null);
  const ordinary = async () => null;
  const own = () => true;

  const tree: RouteObject[] = [
    {
      loader: guard,
      children: [
        { path: "/guarded", loader: guard },
        { path: "/ordinary", loader: ordinary },
        { path: "/own", loader: guard, shouldRevalidate: own },
        { path: "/none" },
      ],
    },
  ];

  it("gives every route with a path-scoped loader the path-change rule, and nothing else", () => {
    const [layout] = withPathScopedRevalidation(tree);
    const [guarded, plain, kept, none] = layout.children ?? [];

    expect(layout.shouldRevalidate).toBe(revalidateOnPathChange);
    expect(guarded.shouldRevalidate).toBe(revalidateOnPathChange);
    expect(plain.shouldRevalidate).toBeUndefined();
    expect(kept.shouldRevalidate).toBe(own);
    expect(none.shouldRevalidate).toBeUndefined();
  });

  it("leaves the route tree it was given untouched", () => {
    withPathScopedRevalidation(tree);

    expect(tree[0].shouldRevalidate).toBeUndefined();
    expect(tree[0].children?.[0].shouldRevalidate).toBeUndefined();
  });
});

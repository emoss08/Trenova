import { installSessionContextHandler } from "@/lib/session-context-handler";
import { useSessionContextStore } from "@/stores/session-context-store";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError, api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { isExplainedError } from "@trenova/shared/lib/explained-errors";
import {
  setSessionContext,
  setSessionContextStatusHandler,
} from "@trenova/shared/lib/session-context";
import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationChangedBanner } from "../organization-changed-banner";
import {
  OrganizationChangedDialog,
  OrganizationChangedDialogHost,
} from "../organization-changed-dialog";

const LOADED = { userId: "usr_1", organizationId: "org_loaded", businessUnitId: "bu_loaded" };
const SWITCHED = { ...LOADED, organizationId: "org_switched", businessUnitId: "bu_switched" };
const TITLE = "Your organization changed in another tab";

function staleRefusal(): Response {
  return new Response(
    JSON.stringify({
      type: "https://trenova.app/problems/session-context-stale",
      title: "Session Changed",
      status: 409,
      detail: "Your session changed in another tab. Reload this page to continue.",
      params: SWITCHED,
    }),
    { status: 409, headers: { "Content-Type": "application/problem+json" } },
  );
}

describe("OrganizationChangedDialogHost", () => {
  beforeEach(() => {
    FakeBroadcastChannel.open = [];
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
    clearCsrfToken();
    setCsrfToken("csrf");
    useSessionContextStore.setState({ organizationChanged: false, keptOpen: false });
    installSessionContextHandler();
    setSessionContext(LOADED);
  });

  afterEach(() => {
    setSessionContext(null);
    setSessionContextStatusHandler(null);
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("opens over the page when a request is refused for another organization, and keeps the caller quiet", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => staleRefusal()),
    );
    render(<OrganizationChangedDialogHost />);
    expect(screen.queryByText(TITLE)).not.toBeInTheDocument();

    let error: unknown;
    await act(async () => {
      error = await api.post("/customers/", { name: "Acme" }).catch((caught: unknown) => caught);
    });

    expect(error).toBeInstanceOf(ApiRequestError);
    expect(isExplainedError(error)).toBe(true);
    expect(await screen.findByText(TITLE)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reload this page" })).toBeInTheDocument();
  });

  it("cannot be dismissed", async () => {
    useSessionContextStore.getState().setOrganizationChanged(true);
    render(<OrganizationChangedDialogHost />);
    expect(await screen.findByText(TITLE)).toBeInTheDocument();

    await userEvent.setup().keyboard("{Escape}");

    expect(screen.getByText(TITLE)).toBeInTheDocument();
    expect(useSessionContextStore.getState().organizationChanged).toBe(true);
  });

  it("closes once the server confirms the session this tab acts for", async () => {
    useSessionContextStore.getState().setOrganizationChanged(true);
    render(<OrganizationChangedDialogHost />);
    expect(await screen.findByText(TITLE)).toBeInTheDocument();

    act(() => {
      setSessionContext(SWITCHED);
    });

    expect(useSessionContextStore.getState().organizationChanged).toBe(false);
  });
});

describe("OrganizationChangedDialog", () => {
  it("offers a reload and a way to keep the page open", async () => {
    const onReload = vi.fn();
    const onKeepOpen = vi.fn();
    render(<OrganizationChangedDialog open onReload={onReload} onKeepOpen={onKeepOpen} />);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Reload this page" }));
    await user.click(screen.getByRole("button", { name: "Keep this page open" }));

    expect(onReload).toHaveBeenCalledTimes(1);
    expect(onKeepOpen).toHaveBeenCalledTimes(1);
  });
});

describe("keeping the page open", () => {
  const BAR =
    "Your organization changed in another tab, so nothing more can be saved on this page.";

  beforeEach(() => {
    useSessionContextStore.setState({ organizationChanged: true, keptOpen: false });
  });

  function renderPage() {
    return render(
      <>
        <OrganizationChangedBanner />
        <OrganizationChangedDialogHost />
      </>,
    );
  }

  it("swaps the prompt for a bar so the person can copy their work", async () => {
    renderPage();
    expect(screen.queryByText(BAR)).not.toBeInTheDocument();

    await userEvent
      .setup()
      .click(await screen.findByRole("button", { name: "Keep this page open" }));

    expect(screen.queryByText(TITLE)).not.toBeInTheDocument();
    expect(screen.getByText(BAR)).toBeInTheDocument();
  });

  it("keeps the prompt closed when later requests are refused too", async () => {
    renderPage();
    await userEvent
      .setup()
      .click(await screen.findByRole("button", { name: "Keep this page open" }));

    act(() => {
      useSessionContextStore.getState().setOrganizationChanged(true);
    });

    expect(screen.queryByText(TITLE)).not.toBeInTheDocument();
    expect(screen.getByText(BAR)).toBeInTheDocument();
  });

  it("clears the bar when the session comes back to this tab's organization", () => {
    useSessionContextStore.setState({ keptOpen: true });
    renderPage();
    expect(screen.getByText(BAR)).toBeInTheDocument();

    act(() => {
      useSessionContextStore.getState().setOrganizationChanged(false);
    });

    expect(screen.queryByText(BAR)).not.toBeInTheDocument();
    expect(useSessionContextStore.getState().keptOpen).toBe(false);
  });
});

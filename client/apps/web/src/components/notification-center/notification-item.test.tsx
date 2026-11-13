import { createDropdownMenuHandle } from "@trenova/shared/components/ui/dropdown-menu";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import type { Notification, NotificationLens } from "@trenova/shared/types/notification";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { NotificationItem } from "./notification-item";
import type { NotificationRowHandlers } from "./use-notification-row-handlers";

const requestGraphQL = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));
vi.mock("@/components/resolved-user-avatar", () => ({
  ResolvedUserAvatar: ({ name }: { name?: string }) => <span data-testid="avatar">{name}</span>,
}));

const NOW = 1_800_000_000;

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "notif_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: "usr_1",
    eventType: "watchtower.assigned",
    priority: "high",
    channel: "user",
    title: "Missed pickup",
    message: "The window closed at 9:00 AM and no arrival is recorded.",
    data: { itemId: "wti_1", link: "/desk/needs?item=wti_1" },
    relatedEntities: null,
    source: "watchtower",
    readAt: null,
    dismissedAt: null,
    createdAt: NOW - 840,
    reason: "direct",
    occurrenceCount: 1,
    surfacesAt: NOW - 840,
    resolvedAt: null,
    resolution: "",
    ...overrides,
  };
}

function handlersOf() {
  return {
    open: vi.fn<NotificationRowHandlers["open"]>(),
    read: vi.fn<NotificationRowHandlers["read"]>(),
    unread: vi.fn<NotificationRowHandlers["unread"]>(),
    done: vi.fn<NotificationRowHandlers["done"]>(),
    later: vi.fn<NotificationRowHandlers["later"]>(),
    wake: vi.fn<NotificationRowHandlers["wake"]>(),
    restore: vi.fn<NotificationRowHandlers["restore"]>(),
    run: vi.fn<NotificationRowHandlers["run"]>(() => Promise.resolve()),
    turnOff: vi.fn<NotificationRowHandlers["turnOff"]>(),
    copyLink: vi.fn<NotificationRowHandlers["copyLink"]>(),
  } satisfies NotificationRowHandlers;
}

function renderRow(
  row: Notification,
  { lens = "forYou", leaving = false }: { lens?: NotificationLens; leaving?: boolean } = {},
) {
  const handlers = handlersOf();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <NotificationItem
        notification={row}
        lens={lens}
        now={NOW}
        leaving={leaving}
        handlers={handlers}
        laterMenu={createDropdownMenuHandle<Notification>()}
        moreMenu={createDropdownMenuHandle<Notification>()}
      />
    </QueryClientProvider>,
  );

  return { handlers, row: screen.getByRole("button", { name: /Missed pickup|mentioned you/ }) };
}

beforeEach(() => {
  requestGraphQL.mockReset();
});

describe("NotificationItem — what it says", () => {
  it("says what happened, when, why the person got it, and that it is unread", () => {
    const { row } = renderRow(notification());

    expect(row).toHaveAccessibleName("Unread: Missed pickup");
    expect(
      screen.getByText("The window closed at 9:00 AM and no arrival is recorded."),
    ).toBeVisible();
    expect(screen.getByText(formatRelativeTime(-840, "narrow"))).toBeVisible();
    expect(screen.getByText("Assigned to you")).toBeVisible();
  });

  it("drops the unread mark once read, and keeps the title as it was", () => {
    const { row } = renderRow(notification({ readAt: NOW - 60 }));

    expect(row).toHaveAccessibleName("Missed pickup");
  });

  it("counts a subject that came up more than once, and says nothing for one that did not", () => {
    renderRow(notification({ occurrenceCount: 14 }));
    expect(screen.getByLabelText("Came up 14 times")).toHaveTextContent("14");
  });

  it("shows no count for a subject that came up once", () => {
    renderRow(notification());
    expect(screen.queryByLabelText(/Came up/)).toBeNull();
  });

  it("gives a team's row its category where its kind has no reason of its own", () => {
    renderRow(notification({ eventType: "watchtower.critical", reason: "team" }));

    expect(screen.getByText("Exceptions")).toBeVisible();
  });

  it("marks a critical notification", () => {
    renderRow(notification({ priority: "critical" }));
    expect(screen.getByText("Critical")).toBeVisible();
  });

  it("says how many days proposals have waited", () => {
    renderRow(
      notification({
        eventType: "agent.proposals_reminder",
        reason: "team",
        data: { waitingHours: 74 },
      }),
    );

    expect(screen.getByText("Waiting 3 days")).toBeVisible();
    expect(screen.getByText("Anyone who can approve")).toBeVisible();
  });
});

describe("NotificationItem — opening it", () => {
  it("opens the record it is about when the row is pressed", () => {
    const row = notification();
    const view = renderRow(row);

    fireEvent.click(view.row);

    expect(view.handlers.open).toHaveBeenCalledWith(row, "/desk/needs?item=wti_1");
    expect(view.handlers.read).not.toHaveBeenCalled();
  });

  it("opens it with Enter while the row has the focus", () => {
    const row = notification();
    const view = renderRow(row);

    fireEvent.keyDown(view.row, { key: "Enter" });

    expect(view.handlers.open).toHaveBeenCalledWith(row, "/desk/needs?item=wti_1");
  });

  it("marks a row with nowhere to go read instead", () => {
    const view = renderRow(
      notification({ eventType: "some.unregistered.event", data: null, source: "system" }),
    );

    fireEvent.click(view.row);

    expect(view.handlers.read).toHaveBeenCalledWith("notif_1");
    expect(view.handlers.open).not.toHaveBeenCalled();
  });
});

describe("NotificationItem — its own actions", () => {
  it("acknowledges the exception from the row without opening it, and holds the button meanwhile", async () => {
    const row = notification();
    const view = renderRow(row);
    let settle: () => void = () => undefined;
    view.handlers.run.mockReturnValue(
      new Promise<void>((resolve) => {
        settle = resolve;
      }),
    );
    const acknowledge = screen.getByRole("button", { name: "Acknowledge" });

    fireEvent.click(acknowledge);

    expect(view.handlers.run).toHaveBeenCalledWith(row, {
      kind: "acknowledge-exception",
      label: "Acknowledge",
      itemId: "wti_1",
    });
    expect(view.handlers.open).not.toHaveBeenCalled();
    await waitFor(() => expect(acknowledge).toBeDisabled());
    expect(acknowledge).toHaveAttribute("aria-busy", "true");

    settle();
    await waitFor(() => expect(acknowledge).toBeEnabled());
  });

  it("opens where a second action leads, as a quieter button", () => {
    const row = notification();
    const view = renderRow(row);

    fireEvent.click(screen.getByRole("button", { name: "Open exception" }));

    expect(view.handlers.open).toHaveBeenCalledTimes(1);
    expect(view.handlers.open).toHaveBeenCalledWith(row, "/desk/needs?item=wti_1", undefined);
  });

  it("tells the handler when opening is all the notification asked for", () => {
    const row = notification({
      eventType: "assistant_reply_ready",
      title: "Missed pickup",
      data: { threadId: "athr_1", status: "Completed", link: "/desk/c/athr_1" },
    });
    const view = renderRow(row);

    fireEvent.click(screen.getByRole("button", { name: "Read the reply" }));

    expect(view.handlers.open).toHaveBeenCalledWith(row, "/desk/c/athr_1", true);
  });

  it("finishes the row from its quick action without opening it", () => {
    const view = renderRow(notification());

    fireEvent.click(screen.getByRole("button", { name: "Done" }));

    expect(view.handlers.done).toHaveBeenCalledWith("notif_1");
    expect(view.handlers.open).not.toHaveBeenCalled();
  });

  it("offers Later and More beside Done on an open row", () => {
    renderRow(notification());

    expect(screen.getByRole("button", { name: "Later" })).toBeVisible();
    expect(screen.getByRole("button", { name: "More" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Move back" })).toBeNull();
  });
});

describe("NotificationItem — the latest times its subject came up", () => {
  const rollup = () =>
    notification({
      eventType: "watchtower.critical",
      reason: "team",
      occurrenceCount: 14,
      data: {
        link: "/desk/needs",
        recent: [
          {
            label: "Mills & Lee LLC, Savannah, GA",
            ref: "S261",
            link: "/desk/needs?item=a",
            at: NOW - 60,
          },
          { label: "Jackson & Jenkins, Memphis, TN", at: NOW - 3_600 },
        ],
      },
    });

  it("keeps them folded until asked for", () => {
    renderRow(rollup());

    expect(screen.getByRole("button", { name: "Latest 2" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    expect(screen.queryByText("Mills & Lee LLC, Savannah, GA")).toBeNull();
  });

  it("lists them with the record's number and time, and opens the one pressed", () => {
    const row = rollup();
    const view = renderRow(row);

    fireEvent.click(screen.getByRole("button", { name: "Latest 2" }));

    const list = screen.getByRole("list");
    expect(within(list).getByText("S261")).toBeVisible();
    expect(within(list).getByText("Jackson & Jenkins, Memphis, TN")).toBeVisible();
    expect(within(list).getByText(formatRelativeTime(-3_600, "narrow"))).toBeVisible();

    fireEvent.click(within(list).getByRole("button", { name: "Mills & Lee LLC, Savannah, GA" }));
    expect(view.handlers.open).toHaveBeenCalledWith(row, "/desk/needs?item=a");
  });

  it("offers no list for a subject with one occurrence on record", () => {
    renderRow(
      notification({ data: { itemId: "wti_1", recent: [{ label: "Only one", at: NOW - 60 }] } }),
    );

    expect(screen.queryByRole("button", { name: /^Latest/ })).toBeNull();
  });
});

describe("NotificationItem — a mention", () => {
  const mention = () =>
    notification({
      eventType: "shipment_comment_mention",
      title: "Dana Whitfield mentioned you",
      message: "@Maya can you confirm the reefer setpoint?",
      data: { authorId: "usr_dana", authorName: "Dana Whitfield" },
      relatedEntities: { shipmentId: "shp_1" },
    });

  it("quotes what was said and shows who said it", () => {
    renderRow(mention());

    expect(screen.getByText("@Maya can you confirm the reefer setpoint?")).toBeVisible();
    expect(screen.getByTestId("avatar")).toHaveTextContent("Dana Whitfield");
    expect(screen.getByText("You were mentioned")).toBeVisible();
  });

  it("sends the reply to the shipment's thread, mentioning the author back, and finishes the row", async () => {
    requestGraphQL.mockResolvedValue({});
    const view = renderRow(mention());
    const box = screen.getByRole("textbox", { name: "Reply to Dana Whitfield…" });

    expect(screen.getByRole("button", { name: "Reply" })).toBeDisabled();
    fireEvent.change(box, { target: { value: "Set to 34°F." } });
    fireEvent.click(screen.getByRole("button", { name: "Reply" }));

    await waitFor(() => expect(view.handlers.done).toHaveBeenCalledWith("notif_1"));
    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "CreateShipmentComment",
        variables: {
          shipmentId: "shp_1",
          input: { comment: "@Dana Whitfield Set to 34°F.", mentionedUserIds: ["usr_dana"] },
        },
      }),
    );
    expect(view.handlers.open).not.toHaveBeenCalled();
  });
});

describe("NotificationItem — set aside", () => {
  const later = () => notification({ surfacesAt: NOW + 17 * 3_600 });

  it("says when it comes back instead of how old it is, and is not marked unread", () => {
    const { row } = renderRow(later(), { lens: "later" });

    expect(screen.getByText(/^Back /)).toBeVisible();
    expect(row).toHaveAccessibleName("Missed pickup");
  });

  it("offers only to bring it back now", () => {
    const view = renderRow(later(), { lens: "later" });

    expect(screen.queryByRole("button", { name: "Acknowledge" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Done" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Bring back now" }));

    expect(view.handlers.wake).toHaveBeenCalledWith("notif_1");
    expect(view.handlers.open).not.toHaveBeenCalled();
  });
});

describe("NotificationItem — finished", () => {
  const finished = (overrides: Partial<Notification> = {}) =>
    notification({ readAt: NOW - 300, dismissedAt: NOW - 300, ...overrides });

  it("says the person marked it done, with the time it was finished", () => {
    renderRow(finished(), { lens: "done" });

    expect(screen.getByText("You marked it done")).toBeVisible();
    expect(screen.getByText(formatRelativeTime(-300, "narrow"))).toBeVisible();
    expect(screen.queryByText("Resolved")).toBeNull();
  });

  it.each([
    ["resolved", "The exception was resolved"],
    ["decided", "The changes were decided"],
    ["cleared", "The condition it reported no longer holds"],
    ["a-code-from-a-newer-server", "It cleared itself"],
  ])("says how it cleared itself for the %s resolution", (resolution, words) => {
    renderRow(finished({ resolvedAt: NOW - 300, resolution }), { lens: "done" });

    expect(screen.getByText("Resolved")).toBeVisible();
    expect(screen.getByText(words)).toBeVisible();
  });

  it("offers only the way back", () => {
    const view = renderRow(finished({ priority: "critical" }), { lens: "done" });

    expect(screen.queryByRole("button", { name: "Acknowledge" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Done" })).toBeNull();
    expect(screen.queryByText("Critical")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Move back" }));

    expect(view.handlers.restore).toHaveBeenCalledWith("notif_1");
  });
});

describe("NotificationItem — leaving the list", () => {
  it("takes a row that is folding away out of the tab order", () => {
    const { row } = renderRow(notification(), { leaving: true });

    expect(row).toHaveAttribute("tabindex", "-1");
  });
});

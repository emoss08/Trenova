import { recordPath } from "@/config/record-links";
import { newShipmentForCustomerPath, SHIPMENTS_PATH } from "@/lib/shipment-links";
import { describe, expect, it } from "vitest";
import {
  ACTION_ASK,
  ACTION_COPY_ID,
  ACTION_COPY_LINK,
  ACTION_NEW_TAB,
  ACTION_OPEN,
  ACTION_PIN,
  buildItemActions,
  CUSTOMER_LEDGER_PATH,
} from "../palette-actions";
import type { PaletteItem } from "../palette-model";
import { actionContext, command, page, record, t } from "./palette-test-fixtures";

function recordItem(overrides: Parameters<typeof record>[0] = {}): PaletteItem {
  const value = record(overrides);
  return { kind: "record", key: `record:${value.entityType}:${value.id}`, record: value };
}

function ids(item: PaletteItem, context = actionContext()): string[] {
  return buildItemActions(item, context, t).map((action) => action.id);
}

function find(item: PaletteItem, id: string, context = actionContext()) {
  return buildItemActions(item, context, t).find((action) => action.id === id);
}

describe("buildItemActions", () => {
  it("opens a shipment first, then asks, links, copies and acts on what it knows", () => {
    const item = recordItem({
      metadata: { bol: "B-1", customerId: "cus_1", customerName: "Acme Foods" },
    });

    expect(buildItemActions(item, actionContext(), t)[0]).toMatchObject({
      id: ACTION_OPEN,
      label: "Open shipment",
      intent: { type: "navigate", href: item.kind === "record" ? item.record.href : "" },
      shortcut: ["↵"],
    });
    expect(ids(item)).toEqual([
      ACTION_OPEN,
      ACTION_ASK,
      ACTION_NEW_TAB,
      ACTION_COPY_LINK,
      ACTION_COPY_ID,
      "open-customer",
      "copy-bol",
    ]);
  });

  it("copies the PRO number from metadata, falling back to the title, and says what it copied", () => {
    expect(
      find(recordItem({ title: "BOL-1", metadata: { proNumber: "PRO-9" } }), ACTION_COPY_ID)
        ?.intent,
    ).toEqual({ type: "copy", text: "PRO-9", what: "pro" });
    expect(find(recordItem({ title: "PRO-1" }), ACTION_COPY_ID)?.intent).toEqual({
      type: "copy",
      text: "PRO-1",
      what: "pro",
    });
  });

  it("copies a customer's or worker's name and a document's file name", () => {
    expect(
      find(recordItem({ entityType: "customer", id: "cus_1", title: "Acme" }), ACTION_COPY_ID)
        ?.intent,
    ).toEqual({ type: "copy", text: "Acme", what: "name" });
    expect(
      find(recordItem({ entityType: "document", id: "doc_1", title: "bol.pdf" }), ACTION_COPY_ID)
        ?.intent,
    ).toEqual({ type: "copy", text: "bol.pdf", what: "file-name" });
  });

  it("offers no BOL copy or customer link when the hit carries neither", () => {
    expect(ids(recordItem())).not.toContain("copy-bol");
    expect(ids(recordItem())).not.toContain("open-customer");
  });

  it("offers to assign a driver only to an unassigned shipment with no driver", () => {
    expect(ids(recordItem({ metadata: { status: "New" } }))).toContain("assign-driver");
    expect(ids(recordItem({ metadata: { status: "PartiallyAssigned" } }))).toContain(
      "assign-driver",
    );
    expect(ids(recordItem({ metadata: { status: "New", driverName: "Dana" } }))).not.toContain(
      "assign-driver",
    );
    expect(ids(recordItem({ metadata: { status: "InTransit" } }))).not.toContain("assign-driver");
  });

  it("asks Nova about the record itself, on the platform's modifier and I", () => {
    expect(find(recordItem({ id: "shp_9", title: "S-9" }), ACTION_ASK)).toMatchObject({
      intent: { type: "ask-about", subject: { type: "shipment", id: "shp_9", label: "S-9" } },
      shortcut: ["⌘", "I"],
    });
  });

  it("leaves out asking Nova for someone who cannot use it", () => {
    expect(ids(recordItem(), actionContext({ canUseAssistant: false }))).not.toContain(ACTION_ASK);
  });

  it("offers the customer ledger only to someone who can reach it", () => {
    const customer = recordItem({ entityType: "customer", id: "cus_1", title: "Acme" });

    expect(
      find(
        customer,
        "customer-ledger",
        actionContext({ canReach: (path) => path === CUSTOMER_LEDGER_PATH }),
      )?.intent,
    ).toEqual({ type: "navigate", href: `${CUSTOMER_LEDGER_PATH}?customerId=cus_1` });
    expect(ids(customer, actionContext({ canReach: () => false }))).not.toContain(
      "customer-ledger",
    );
  });

  it("starts a shipment for a customer only for someone who may create one and reach the board", () => {
    const customer = recordItem({ entityType: "customer", id: "cus_1", title: "Acme" });

    expect(find(customer, "new-shipment-for-customer")).toMatchObject({
      label: "New shipment for {0}|Acme",
      intent: { type: "navigate", href: newShipmentForCustomerPath("cus_1") },
    });
    expect(ids(customer, actionContext({ canCreateShipment: false }))).not.toContain(
      "new-shipment-for-customer",
    );
    expect(
      ids(customer, actionContext({ canReach: (path) => path !== SHIPMENTS_PATH })),
    ).not.toContain("new-shipment-for-customer");
  });

  it("opens a worker's tabs through the record link registry", () => {
    const worker = recordItem({ entityType: "worker", id: "wrk_1", title: "Dana Ruiz" });

    expect(find(worker, "worker-pto")?.intent).toEqual({
      type: "navigate",
      href: recordPath("worker", "wrk_1", { tab: "pto" }),
    });
    expect(find(worker, "worker-compliance")?.intent).toEqual({
      type: "navigate",
      href: recordPath("worker", "wrk_1", { tab: "compliance" }),
    });
  });

  it("opens a worker's current load only when both its id and number are known", () => {
    const onLoad = recordItem({
      entityType: "worker",
      id: "wrk_1",
      title: "Dana Ruiz",
      metadata: { currentLoad: "S-1", currentLoadId: "shp_1" },
    });
    const idOnly = recordItem({
      entityType: "worker",
      id: "wrk_1",
      title: "Dana Ruiz",
      metadata: { currentLoadId: "shp_1" },
    });

    expect(find(onLoad, "open-current-load")).toMatchObject({
      label: "Open {0}|S-1",
      intent: { type: "navigate", href: recordPath("shipment", "shp_1") },
    });
    expect(ids(idOnly)).not.toContain("open-current-load");
  });

  it("previews a document first rather than opening the record it is attached to", () => {
    const document = recordItem({
      entityType: "document",
      id: "doc_1",
      title: "bol.pdf",
      href: "/parent",
      metadata: { attachedLabel: "S-240193" },
    });

    expect(buildItemActions(document, actionContext(), t)[0]).toMatchObject({
      label: "Preview document",
      intent: { type: "open-document", documentId: "doc_1", disposition: "view" },
    });
    expect(find(document, "document-download")?.intent).toEqual({
      type: "open-document",
      documentId: "doc_1",
      disposition: "download",
    });
    expect(find(document, "document-record")).toMatchObject({
      label: "Open {0}|S-240193",
      intent: { type: "navigate", href: "/parent" },
    });
  });

  it("offers to unpin a page that is pinned and to pin one that is not, on the modifier and P", () => {
    const item: PaletteItem = { kind: "page", key: "page:/billing/invoices", page: page() };
    const pin = (pinned: boolean) =>
      find(
        item,
        ACTION_PIN,
        actionContext({ pinnedUrls: new Set(pinned ? ["/billing/invoices"] : []) }),
      );

    expect(pin(false)).toMatchObject({
      label: "Pin to home",
      intent: { type: "toggle-pin", pageUrl: "/billing/invoices", pageTitle: "Invoices" },
      shortcut: ["⌘", "P"],
    });
    expect(pin(true)?.label).toBe("Unpin from home");
  });

  it("lists the commands that start something on a page after the page's own actions", () => {
    const create = command({
      id: "quick:create-invoice",
      label: "New invoice",
      shortcut: ["⌘", "N"],
    });
    const item: PaletteItem = { kind: "page", key: "page:/billing/invoices", page: page() };

    const actions = buildItemActions(
      item,
      actionContext({
        relatedCommands: (href) => (href === "/billing/invoices" ? [create] : []),
      }),
      t,
    );

    expect(actions.map((action) => action.id)).toEqual([
      ACTION_OPEN,
      ACTION_NEW_TAB,
      ACTION_COPY_LINK,
      ACTION_PIN,
      "related:quick:create-invoice",
    ]);
    expect(actions.at(-1)).toMatchObject({
      label: "New invoice",
      intent: create.intent,
      shortcut: ["⌘", "N"],
    });
  });

  it("runs a command as its only action", () => {
    const item: PaletteItem = { kind: "command", key: "command:x", command: command() };

    expect(buildItemActions(item, actionContext(), t)).toEqual([
      expect.objectContaining({ id: ACTION_OPEN, intent: command().intent, shortcut: ["↵"] }),
    ]);
  });

  it("opens a queue and links to it from an attention row", () => {
    const item: PaletteItem = {
      kind: "attention",
      key: "attention:billing",
      attention: {
        key: "billing",
        label: "Billing queue",
        module: "Billing",
        count: 6,
        href: "/billing/queue",
        tone: "warning",
        reason: "",
      },
    };

    expect(ids(item)).toEqual([ACTION_OPEN, ACTION_NEW_TAB, ACTION_COPY_LINK]);
    expect(find(item, ACTION_COPY_LINK)?.intent).toEqual({
      type: "copy-link",
      href: "/billing/queue",
    });
  });

  it("puts copying a number on Alt, clear of the browser's own Shift+C inspector", () => {
    const copyId = (mac: boolean) =>
      find(recordItem(), ACTION_COPY_ID, actionContext({ mac }))?.shortcut;

    expect(copyId(true)).toEqual(["⌥", "C"]);
    expect(copyId(false)).toEqual(["Alt", "C"]);
  });

  it("spells the modifier the way the platform does", () => {
    const shortcut = (id: string, mac: boolean) =>
      find(recordItem(), id, actionContext({ mac }))?.shortcut;

    expect(shortcut(ACTION_NEW_TAB, true)).toEqual(["⌘", "↵"]);
    expect(shortcut(ACTION_NEW_TAB, false)).toEqual(["Ctrl", "↵"]);
    expect(shortcut(ACTION_COPY_LINK, false)).toEqual(["Ctrl", "L"]);
  });
});

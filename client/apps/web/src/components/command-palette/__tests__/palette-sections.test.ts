import type { GlobalSearchGroup } from "@/services/global-search";
import { describe, expect, it } from "vitest";
import {
  buildHomeSections,
  buildSearchSections,
  flattenSections,
  recordsFromGroups,
  type HomeInput,
  type RemoteState,
  type SearchInput,
} from "../palette-sections";
import { command, page, record, t } from "./palette-test-fixtures";

const IDLE_REMOTE: RemoteState = { groups: [], loading: false, error: false, ready: false };

function home(overrides: Partial<HomeInput> = {}): HomeInput {
  return {
    recentRecords: [],
    attention: { rows: [], loading: false },
    notifications: { items: [], unread: 0, loading: false },
    pinnedPages: { pages: [], loading: false },
    recentPages: [],
    suggested: [],
    ...overrides,
  };
}

function search(overrides: Partial<SearchInput> = {}): SearchInput {
  return {
    query: "",
    scope: "all",
    question: null,
    remote: IDLE_REMOTE,
    pages: [],
    commands: [],
    recentRecords: [],
    ...overrides,
  };
}

// Shaped like the Go handler's JSON: `subtitle` and `metadata` are omitempty,
// so a hit may carry neither, and a group only exists when it has hits.
const SHIPMENT_GROUP: GlobalSearchGroup = {
  entityType: "shipment",
  label: "Shipments",
  hits: [
    {
      id: "shp_1",
      entityType: "shipment",
      title: "BOL-77",
      href: "/shipment-management/shipments?expanded=shp_1",
      metadata: { proNumber: "PRO-1001", status: "InTransit", bol: "BOL-77" },
    },
    {
      id: "shp_2",
      entityType: "shipment",
      title: "PRO-1002",
      href: "/shipment-management/shipments?expanded=shp_2",
    },
  ],
};

const CUSTOMER_GROUP: GlobalSearchGroup = {
  entityType: "customer",
  label: "Customers",
  hits: [
    {
      id: "cus_1",
      entityType: "customer",
      title: "Acme Freight",
      subtitle: "ACME",
      href: "/billing/configuration-files/customers?panelEntityId=cus_1",
      metadata: { status: "Active" },
    },
  ],
};

describe("recordsFromGroups", () => {
  it("prefers the enriched PRO number over the indexed title, and survives absent metadata", () => {
    const byType = recordsFromGroups([SHIPMENT_GROUP]);
    const shipments = byType.get("shipment") ?? [];

    expect(shipments.map((entry) => entry.title)).toEqual(["PRO-1001", "PRO-1002"]);
    expect(shipments[1]?.metadata).toEqual({});
    expect(shipments[1]?.subtitle).toBeUndefined();
  });

  it("drops a group whose type the palette does not know rather than guessing", () => {
    const byType = recordsFromGroups([
      {
        entityType: "tractor",
        label: "Tractors",
        hits: [{ ...CUSTOMER_GROUP.hits[0]!, entityType: "tractor" }],
      },
    ]);

    expect(byType.size).toBe(0);
  });
});

describe("buildHomeSections", () => {
  it("shows nothing but the sections that have something, in a fixed order", () => {
    const sections = buildHomeSections(
      home({
        recentRecords: [record()],
        suggested: [command()],
      }),
      t,
    );

    expect(sections.map((section) => section.id)).toEqual(["recent-records", "suggested"]);
  });

  it("keeps a loading section with placeholder rows until its data lands", () => {
    const sections = buildHomeSections(home({ attention: { rows: [], loading: true } }), t);

    expect(sections).toHaveLength(1);
    expect(sections[0]).toMatchObject({ id: "attention", loading: true, placeholderRows: 2 });
  });

  it("does not list a pinned page again under recently visited", () => {
    const pinned = page({ href: "/billing/invoices" });
    const other = page({ id: "hr", title: "Workers", href: "/hr/workers" });
    const sections = buildHomeSections(
      home({
        pinnedPages: { pages: [pinned], loading: false },
        recentPages: [pinned, other],
      }),
      t,
    );

    const recent = sections.find((section) => section.id === "recent-pages");
    expect(recent?.items.map((item) => item.key)).toEqual(["recent-page:/hr/workers"]);
  });

  it("reports every unread notification in the count even when it shows only a few", () => {
    const sections = buildHomeSections(
      home({
        notifications: {
          items: [
            {
              href: null,
              notification: {
                id: "ntf_1",
                organizationId: "org",
                businessUnitId: null,
                targetUserId: null,
                eventType: "x",
                priority: "medium",
                channel: "user",
                title: "Hello",
                message: "",
                data: null,
                relatedEntities: null,
                source: "system",
                readAt: null,
                dismissedAt: null,
                createdAt: 1,
              },
            },
          ],
          unread: 14,
          loading: false,
        },
      }),
      t,
    );

    expect(sections[0]).toMatchObject({ id: "notifications", count: 14 });
    expect(sections[0]?.items).toHaveLength(1);
  });
});

describe("buildSearchSections", () => {
  const pages = [
    page(),
    page({
      id: "shipments",
      title: "Shipments",
      trail: "Shipments > Shipments",
      href: "/shipment-management/shipments",
      module: "Shipments",
      keywords: ["Shipments"],
    }),
  ];
  const commands = [command()];

  it("puts an obvious page first as the top hit and does not repeat it under pages", () => {
    const sections = buildSearchSections(search({ query: "invoices", pages, commands }), t);

    expect(sections[0]?.id).toBe("top-hit");
    expect(sections[0]?.items[0]?.key).toBe("page:/billing/invoices");
    const pagesSection = sections.find((section) => section.id === "pages");
    expect(pagesSection?.items.some((item) => item.key === "page:/billing/invoices")).toBeFalsy();
  });

  it("orders ask, top hit, records, pages, commands", () => {
    const sections = buildSearchSections(
      search({
        query: "ship",
        question: "ship?",
        pages,
        commands,
        remote: { groups: [SHIPMENT_GROUP], loading: false, error: false, ready: true },
      }),
      t,
    );

    expect(sections.map((section) => section.id)).toEqual([
      "ask",
      "top-hit",
      "records:shipment",
      "commands",
    ]);
  });

  it("shows one loading block while the first results are on their way", () => {
    const sections = buildSearchSections(
      search({ query: "acme", remote: { groups: [], loading: true, error: false, ready: true } }),
      t,
    );

    expect(sections).toEqual([expect.objectContaining({ id: "records-loading", loading: true })]);
  });

  it("keeps the last results on screen while the next query is in flight", () => {
    const sections = buildSearchSections(
      search({
        query: "acme",
        remote: { groups: [CUSTOMER_GROUP], loading: true, error: false, ready: true },
      }),
      t,
    );

    expect(flattenSections(sections).map((item) => item.key)).toEqual(["record:customer:cus_1"]);
  });

  it("reports a failed search inside its own section and leaves the rest alone", () => {
    const sections = buildSearchSections(
      search({
        query: "invoices",
        pages,
        remote: { groups: [], loading: false, error: true, ready: true },
      }),
      t,
    );

    expect(sections.map((section) => section.id)).toEqual(["top-hit", "records-error"]);
  });

  it("does not surface a page because its breadcrumb happens to hold the letters in order", () => {
    const sections = buildSearchSections(
      search({
        query: "acme",
        pages: [
          page({ title: "Console", trail: "Dispatch management > Console", keywords: ["Console"] }),
        ],
      }),
      t,
    );

    expect(sections).toEqual([]);
  });

  it("does not search records for a single character", () => {
    const sections = buildSearchSections(search({ query: "a", remote: IDLE_REMOTE }), t);

    expect(sections.some((section) => section.id.startsWith("records"))).toBe(false);
  });

  it("offers recent records of the scoped type before a query is long enough to send", () => {
    const sections = buildSearchSections(
      search({
        query: "",
        scope: "customer",
        recentRecords: [record(), record({ entityType: "customer", id: "cus_9", title: "Globex" })],
      }),
      t,
    );

    expect(flattenSections(sections).map((item) => item.key)).toEqual([
      "recent-record:customer:cus_9",
    ]);
  });

  it("limits a record scope to that type", () => {
    const sections = buildSearchSections(
      search({
        query: "acme",
        scope: "customer",
        remote: {
          groups: [SHIPMENT_GROUP, CUSTOMER_GROUP],
          loading: false,
          error: false,
          ready: true,
        },
      }),
      t,
    );

    expect(sections.map((section) => section.id)).toEqual(["records:customer"]);
  });

  it("groups every page by module when the pages scope has no query", () => {
    const sections = buildSearchSections(search({ scope: "pages", pages }), t);

    expect(sections.map((section) => section.heading)).toEqual(["Billing", "Shipments"]);
  });

  it("groups commands in the fixed group order", () => {
    const sections = buildSearchSections(
      search({
        scope: "commands",
        commands: [command({ id: "app:sign-out", label: "Log out", group: "account" }), command()],
      }),
      t,
    );

    expect(sections.map((section) => section.id)).toEqual(["commands:create", "commands:account"]);
  });
});

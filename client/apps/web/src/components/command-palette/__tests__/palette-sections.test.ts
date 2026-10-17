import { adminLinks, navigationConfig } from "@/config/navigation.config";
import type { GlobalSearchGroup } from "@/services/global-search";
import { describe, expect, it } from "vitest";
import type { PaletteItem, PalettePage, PaletteSection } from "../palette-model";
import {
  buildHomeSections,
  buildSearchSections,
  buildSubjectSections,
  flattenSections,
  MAX_ASK_MENTIONS,
  rankLocal,
  recordsFromGroups,
  recordsNamedIn,
  scopeCounts,
  type HomeInput,
  type RemoteState,
  type SearchInput,
} from "../palette-sections";
import { buildRouteCommandGroups } from "../route-command-data";
import { command, page, record, t } from "./palette-test-fixtures";

const IDLE_REMOTE: RemoteState = { groups: [], loading: false, error: false, ready: false };

function settled(groups: GlobalSearchGroup[]): RemoteState {
  return { groups, loading: false, error: false, ready: true };
}

function home(overrides: Partial<HomeInput> = {}): HomeInput {
  return {
    recentRecords: [],
    attention: { rows: [], loading: false },
    pinnedPages: { pages: [], loading: false },
    quickActions: [],
    ...overrides,
  };
}

function search(
  overrides: Partial<Omit<SearchInput, "ranking">> & {
    pages?: PalettePage[];
    commands?: ReturnType<typeof command>[];
  } = {},
): SearchInput {
  const { pages = [], commands = [], ...rest } = overrides;
  const query = rest.query ?? "";
  return {
    query,
    recordQuery: rest.recordQuery ?? query,
    scope: "all",
    question: null,
    remote: IDLE_REMOTE,
    recentRecords: [],
    ranking: rankLocal(query, pages, commands, t),
    ...rest,
  };
}

function ids(sections: readonly PaletteSection[]): string[] {
  return sections.map((entry) => entry.id);
}

function keys(items: readonly PaletteItem[]): string[] {
  return items.map((item) => item.key);
}

function sectionOf(sections: readonly PaletteSection[], id: string): PaletteSection | undefined {
  return sections.find((entry) => entry.id === id);
}

// Shaped like the Go handler's JSON: `subtitle`, `metadata` and `total` are
// omitempty, so a hit may carry neither, and a group only exists when it has hits.
const SHIPMENT_GROUP: GlobalSearchGroup = {
  entityType: "shipment",
  label: "Shipments",
  total: 7,
  hits: [
    {
      id: "shp_1",
      entityType: "shipment",
      title: "BOL-77",
      href: "/shipment-management/shipments?expanded=shp_1",
      metadata: {
        proNumber: "S-1001",
        status: "InTransit",
        bol: "77",
        customerName: "Acme Foods",
        origin: "Dallas, TX",
        destination: "Atlanta, GA",
      },
    },
    {
      id: "shp_2",
      entityType: "shipment",
      title: "S-1002",
      href: "/shipment-management/shipments?expanded=shp_2",
      metadata: { customerName: "Acme Foods" },
    },
    {
      id: "shp_3",
      entityType: "shipment",
      title: "S-1003",
      href: "/shipment-management/shipments?expanded=shp_3",
      metadata: { customerName: "Acme Foods" },
    },
    {
      id: "shp_4",
      entityType: "shipment",
      title: "S-1004",
      href: "/shipment-management/shipments?expanded=shp_4",
      metadata: { customerName: "Acme Foods" },
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
      title: "Acme Foods",
      subtitle: "ACME",
      href: "/billing/configuration-files/customers?panelEntityId=cus_1",
      metadata: { status: "Active", code: "ACME" },
    },
  ],
};

describe("recordsNamedIn", () => {
  it("hands Nova the records the question names, and not the looser matches beside them", () => {
    expect(
      recordsNamedIn("Why is S-1001 running late for Acme Foods?", [
        SHIPMENT_GROUP,
        CUSTOMER_GROUP,
      ]),
    ).toEqual([
      { type: "shipment", id: "shp_1", label: "S-1001" },
      { type: "customer", id: "cus_1", label: "Acme Foods" },
    ]);
  });

  it("reads a name in any case and with a possessive, but never inside a longer word", () => {
    expect(recordsNamedIn("what is acme foods's balance?", [CUSTOMER_GROUP])).toEqual([
      { type: "customer", id: "cus_1", label: "Acme Foods" },
    ]);
    expect(recordsNamedIn("Tell me about Acme Foodservice", [CUSTOMER_GROUP])).toEqual([]);
    expect(recordsNamedIn("Tell me about S-10011", [SHIPMENT_GROUP])).toEqual([]);
  });

  it("names each record once even when two groups return it", () => {
    expect(
      recordsNamedIn("Acme Foods?", [CUSTOMER_GROUP, { ...CUSTOMER_GROUP, label: "Again" }]),
    ).toEqual([{ type: "customer", id: "cus_1", label: "Acme Foods" }]);
  });

  it("sends no more records than the server takes for one question", () => {
    const many: GlobalSearchGroup = {
      entityType: "shipment",
      label: "Shipments",
      hits: Array.from({ length: MAX_ASK_MENTIONS + 2 }, (_, index) => ({
        id: `shp_${index}`,
        entityType: "shipment",
        title: `S-20${String(index).padStart(2, "0")}`,
        href: `/shipments/shp_${index}`,
      })),
    };
    const question = many.hits.map((hit) => hit.title).join(", ");

    expect(recordsNamedIn(question, [many])).toHaveLength(MAX_ASK_MENTIONS);
  });

  it("skips a title too short to be told from an ordinary word", () => {
    const short: GlobalSearchGroup = {
      entityType: "worker",
      label: "Workers",
      hits: [{ id: "wrk_1", entityType: "worker", title: "Al", href: "/workers/wrk_1" }],
    };

    expect(recordsNamedIn("Can Al take a load today?", [short])).toEqual([]);
  });
});

describe("recordsFromGroups", () => {
  it("prefers the enriched PRO number over the indexed title, and survives absent metadata", () => {
    const shipments = recordsFromGroups([
      {
        entityType: "shipment",
        label: "Shipments",
        hits: [
          SHIPMENT_GROUP.hits[0]!,
          { id: "shp_9", entityType: "shipment", title: "S-9", href: "/x" },
        ],
      },
    ]).get("shipment");

    expect(shipments?.records.map((entry) => entry.title)).toEqual(["S-1001", "S-9"]);
    expect(shipments?.records[1]?.metadata).toEqual({});
    expect(shipments?.records[1]?.subtitle).toBeUndefined();
  });

  it("gives the same records for the same answer, so rows rebuilt per key keep their record", () => {
    const groups = [SHIPMENT_GROUP];

    expect(recordsFromGroups(groups).get("shipment")?.records[0]).toBe(
      recordsFromGroups(groups).get("shipment")?.records[0],
    );
    expect(recordsFromGroups([SHIPMENT_GROUP]).get("shipment")?.records[0]).not.toBe(
      recordsFromGroups(groups).get("shipment")?.records[0],
    );
  });

  it("takes the index's total when it is larger than the hits it returned", () => {
    expect(recordsFromGroups([SHIPMENT_GROUP]).get("shipment")?.total).toBe(7);
  });

  it("counts the hits themselves when the server sent no total", () => {
    expect(recordsFromGroups([CUSTOMER_GROUP]).get("customer")?.total).toBe(1);
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
  it("draws Jump back in, Needs you, Pinned and Quick actions in that order, skipping empty ones", () => {
    const sections = buildHomeSections(
      home({
        recentRecords: [{ record: record(), visitedAt: 100 }],
        pinnedPages: { pages: [page()], loading: false },
        quickActions: [command()],
      }),
      t,
    );

    expect(ids(sections)).toEqual(["recent-records", "pinned", "quick-actions"]);
    expect(sections.map((entry) => entry.heading)).toEqual([
      "Jump back in",
      "Pinned",
      "Quick actions",
    ]);
  });

  it("carries when each record was opened, for the relative time on its row", () => {
    const [jump] = buildHomeSections(
      home({ recentRecords: [{ record: record(), visitedAt: 1_700_000_000_000 }] }),
      t,
    );

    expect(jump?.items[0]).toMatchObject({
      kind: "record",
      key: "recent-record:shipment:shp_1",
      visitedAt: 1_700_000_000_000,
    });
  });

  it("shows only the five records opened last", () => {
    const recent = Array.from({ length: 7 }, (_, index) => ({
      record: record({ id: `shp_${index}` }),
      visitedAt: index,
    }));

    const [jump] = buildHomeSections(home({ recentRecords: recent }), t);

    expect(jump?.items).toHaveLength(5);
  });

  it("keeps a loading section with placeholder rows until its data lands", () => {
    const sections = buildHomeSections(home({ attention: { rows: [], loading: true } }), t);

    expect(sections).toEqual([
      expect.objectContaining({ id: "attention", loading: true, placeholderRows: 2, items: [] }),
    ]);
  });

  it("lists what is waiting on the person as attention rows", () => {
    const sections = buildHomeSections(
      home({
        attention: {
          rows: [
            {
              key: "billing",
              label: "Billing queue",
              module: "Billing",
              count: 6,
              href: "/billing/queue",
              tone: "warning",
              reason: "",
            },
          ],
          loading: false,
        },
      }),
      t,
    );

    expect(sections[0]).toMatchObject({ id: "attention", heading: "Needs you" });
    expect(keys(sections[0]!.items)).toEqual(["attention:billing"]);
  });
});

describe("buildSearchSections", () => {
  const shipments = page({
    id: "shipments",
    title: "Shipments",
    trail: ["Shipment management"],
    href: "/shipment-management/shipments",
    keywords: ["loads"],
  });
  const invoices = page();

  it("puts a record that is clearly the answer first as the top hit, not again in its group", () => {
    const { sections } = buildSearchSections(
      search({ query: "acme", remote: settled([SHIPMENT_GROUP, CUSTOMER_GROUP]) }),
      t,
    );

    expect(ids(sections)).toEqual(["top-hit", "records:shipment", "ask-fallback"]);
    expect(keys(sectionOf(sections, "top-hit")!.items)).toEqual(["record:customer:cus_1"]);
    expect(sectionOf(sections, "records:customer")).toBeUndefined();
  });

  it("caps a record group at three and offers the rest through See all", () => {
    const { sections } = buildSearchSections(
      search({ query: "acme", remote: settled([SHIPMENT_GROUP, CUSTOMER_GROUP]) }),
      t,
    );
    const group = sectionOf(sections, "records:shipment")!;

    expect(group.items).toHaveLength(3);
    expect(group).toMatchObject({ count: 7, seeAll: "shipment" });
  });

  it("names the field a record matched on so the row can mark it", () => {
    const { sections } = buildSearchSections(
      search({ query: "acme", remote: settled([SHIPMENT_GROUP, CUSTOMER_GROUP]) }),
      t,
    );
    const first = sectionOf(sections, "records:shipment")!.items[0];

    expect(first?.kind === "record" && first.match?.highlights).toEqual({ customer: [[0, 4]] });
  });

  it("matches a page through its keyword and tags the keyword as the source", () => {
    const { sections } = buildSearchSections(search({ query: "loads", pages: [shipments] }), t);
    const hit = flattenSections(sections).find((item) => item.kind === "page");

    expect(hit?.kind === "page" && hit.match?.via).toEqual({
      label: "Keyword",
      text: "loads",
      ranges: [[0, 5]],
    });
  });

  it("always ends a search with a way to ask Nova", () => {
    const { sections } = buildSearchSections(search({ query: "invoices", pages: [invoices] }), t);

    expect(sections.at(-1)).toMatchObject({ id: "ask-fallback", heading: "Nova" });
    expect(sections.at(-1)?.items[0]).toMatchObject({ kind: "ask", question: "invoices" });
  });

  it("asks Nova first, and only once, when the query reads as a question", () => {
    const { sections } = buildSearchSections(
      search({ query: "why is S-1001 late?", question: "why is S-1001 late?" }),
      t,
    );

    expect(sections[0]).toMatchObject({ id: "nova" });
    expect(flattenSections(sections).filter((item) => item.kind === "ask")).toHaveLength(1);
  });

  it("lists the records a question's search words found under Nova, ranked by those words", () => {
    const { sections } = buildSearchSections(
      search({
        query: "Tell me about S-1002?",
        recordQuery: "S-1002",
        question: "Tell me about S-1002?",
        remote: settled([SHIPMENT_GROUP]),
      }),
      t,
    );

    expect(sections[0]).toMatchObject({ id: "nova" });
    expect(sections[1]).toMatchObject({ id: "top-hit" });
    expect(sections[1]?.items[0]).toMatchObject({ kind: "record", record: { id: "shp_2" } });
    expect(flattenSections(sections).filter((item) => item.kind === "ask")).toHaveLength(1);
  });

  it("reports nothing found only once the search has settled", () => {
    const loading = buildSearchSections(
      search({ query: "zzqx", remote: { groups: [], loading: true, error: false, ready: true } }),
      t,
    );
    const done = buildSearchSections(search({ query: "zzqx", remote: settled([]) }), t);

    expect(loading.empty).toBe(false);
    expect(done.empty).toBe(true);
    expect(sectionOf(done.sections, "ask-fallback")?.heading).toBe("Try asking instead");
  });

  it("shows placeholder rows while records load and an error row when they fail", () => {
    const loading = buildSearchSections(
      search({ query: "ac", remote: { groups: [], loading: true, error: false, ready: true } }),
      t,
    );
    const failed = buildSearchSections(
      search({ query: "ac", remote: { groups: [], loading: false, error: true, ready: true } }),
      t,
    );

    expect(sectionOf(loading.sections, "records-loading")).toMatchObject({ loading: true });
    expect(sectionOf(failed.sections, "records-error")).toMatchObject({ error: true });
  });

  it("browses a scope with nothing typed: the ones opened last, then the rest", () => {
    const recent = recordsFromGroups([SHIPMENT_GROUP]).get("shipment")!.records[1]!;
    const { sections } = buildSearchSections(
      search({
        scope: "shipment",
        remote: settled([SHIPMENT_GROUP]),
        recentRecords: [{ record: recent, visitedAt: 5 }],
      }),
      t,
    );

    expect(ids(sections)).toEqual(["recent-records", "records:shipment"]);
    expect(keys(sections[0]!.items)).toEqual(["recent-record:shipment:shp_2"]);
    expect(keys(sections[1]!.items)).toEqual([
      "record:shipment:shp_1",
      "record:shipment:shp_3",
      "record:shipment:shp_4",
    ]);
    expect(sections[1]?.count).toBe(6);
  });

  it("offers to search everywhere when a scope finds nothing", () => {
    const { sections, empty } = buildSearchSections(
      search({ query: "zzqx", scope: "customer", remote: settled([]) }),
      t,
    );

    expect(empty).toBe(true);
    expect(sections.at(-1)?.items[0]).toMatchObject({ kind: "widen", query: "zzqx" });
  });

  it("browses commands grouped when the commands scope has nothing typed", () => {
    const { sections } = buildSearchSections(
      search({
        scope: "commands",
        commands: [
          command(),
          command({ id: "app:theme-dark", label: "Use dark theme", group: "appearance" }),
        ],
      }),
      t,
    );

    expect(ids(sections)).toEqual(["commands:create", "commands:appearance"]);
  });
});

describe("matching 'ai' against the real navigation", () => {
  const pages: PalettePage[] = buildRouteCommandGroups(
    navigationConfig.modules,
    adminLinks,
  ).flatMap((group) =>
    group.items.map((item) => ({
      id: item.id,
      title: item.title,
      trail: item.trail,
      href: item.href,
      icon: item.icon,
      keywords: item.keywords,
      module: group.label,
    })),
  );
  const ranked = rankLocal("ai", pages, [], t).pages;
  const titles = ranked.map(({ entry }) => entry.title);

  it("does not find 'ai' inside a word", () => {
    expect(titles).not.toContain("Service failures");
    expect(titles).not.toContain("Trailers");
    expect(titles).not.toContain("Email logs");
  });

  it("finds AI control by its title, ahead of anything matched another way", () => {
    expect(ranked[0]?.entry.title).toBe("AI control");
    expect(ranked[0]?.match.highlights).toEqual({ title: [[0, 2]] });
  });

  it("finds Document extraction through its keyword and says so", () => {
    const hit = ranked.find(({ entry }) => entry.title === "Document extraction");

    expect(hit?.match.via).toEqual({ label: "Keyword", text: "ai", ranges: [[0, 2]] });
  });
});

describe("matching a worker by name", () => {
  it("finds Aisha Grant for 'ai' at the start of her name", () => {
    const { sections } = buildSearchSections(
      search({
        query: "ai",
        remote: settled([
          {
            entityType: "worker",
            label: "Workers",
            hits: [
              { id: "wrk_1", entityType: "worker", title: "Aisha Grant", href: "/hr/workers" },
              { id: "wrk_2", entityType: "worker", title: "Blaine Ortiz", href: "/hr/workers" },
            ],
          },
        ]),
      }),
      t,
    );
    const top = sectionOf(sections, "top-hit")?.items[0];

    expect(top?.key).toBe("record:worker:wrk_1");
    expect(top?.kind === "record" && top.match?.highlights).toEqual({ title: [[0, 2]] });
  });
});

describe("buildSubjectSections", () => {
  const subject = { type: "shipment", id: "shp_1", label: "S-1001" };

  it("offers three starter questions about the record before anything is typed", () => {
    const [ask] = buildSubjectSections(subject, "", null, t);

    expect(ask?.heading).toBe("Ask Nova about {0}|S-1001");
    expect(ask?.items).toHaveLength(3);
    expect(ask?.items.every((item) => item.kind === "ask" && item.subject === subject)).toBe(true);
  });

  it("drops the answered question and offers the rest under Ask something else", () => {
    const [first] = buildSubjectSections(subject, "", null, t);
    const answered = first!.items[0]!.kind === "ask" ? first!.items[0]!.question : "";

    const [rest] = buildSubjectSections(subject, "", answered, t);

    expect(rest?.heading).toBe("Ask something else");
    expect(rest?.items).toHaveLength(2);
  });

  it("asks exactly what was typed", () => {
    const [ask] = buildSubjectSections(subject, "who signed the POD?", null, t);

    expect(ask?.items).toEqual([expect.objectContaining({ question: "who signed the POD?" })]);
  });
});

describe("scopeCounts", () => {
  const ranking = rankLocal("acme", [], [], t);

  it("counts each record type from the index's totals once the search settles", () => {
    expect(scopeCounts(ranking, settled([SHIPMENT_GROUP, CUSTOMER_GROUP]))).toEqual({
      all: 8,
      shipment: 7,
      customer: 1,
      worker: 0,
      document: 0,
      pages: 0,
      commands: 0,
    });
  });

  it("leaves record types uncounted while the search is still running", () => {
    const counts = scopeCounts(ranking, {
      groups: [SHIPMENT_GROUP],
      loading: true,
      error: false,
      ready: true,
    });

    expect(counts.shipment).toBeUndefined();
  });

  it("reads zero for the scopes command mode never searched", () => {
    const theme = rankLocal(
      "theme",
      [page({ title: "Theme settings" })],
      [command({ id: "app:theme-dark", label: "Use dark theme", keywords: ["theme"] })],
      t,
    );

    expect(scopeCounts(theme, IDLE_REMOTE, "commands")).toEqual({
      all: 1,
      pages: 0,
      commands: 1,
      shipment: 0,
      customer: 0,
      worker: 0,
      document: 0,
    });
  });

  it("counts the records a question's search words found", () => {
    expect(scopeCounts(ranking, settled([SHIPMENT_GROUP])).shipment).toBe(7);
  });
});

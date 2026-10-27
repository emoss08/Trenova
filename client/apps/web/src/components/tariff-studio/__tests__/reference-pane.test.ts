import { translate } from "@trenova/shared/i18n/runtime";
import { rulesSync, type RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffCatalog } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { dataEntries, languageEntries, libraryEntries } from "../reference-pane";

const message = (text: string) => ({ message: text });

// A cut of what `tariff.catalog` answers, in the shape src/tariff/check/catalog.rs writes: the
// `bill` root after the schema's roots with its fields and `charge` method, ordered and unordered
// enums, the `Time`, `FreightClass` and `Shipment` receivers, the `Time` type, `std.tiers`, and a
// schema field whose descriptor is an optional ordered enum.
const CATALOG = {
  modules: [
    {
      path: "std.tiers",
      doc: message("Graduated (progressive) pricing."),
      declarations: [
        {
          kind: "fn",
          name: "byDistance",
          signature: "fn byDistance(tiers: List<DistanceTier>, distance: Distance) -> Money",
          doc: message("What `distance` costs at graduated `tiers`."),
        },
      ],
    },
  ],
  types: [{ name: "Time", doc: message("A time of day, with no date and no zone: `@17:30`.") }],
  methods: [
    {
      receiver: "Time",
      name: "hour",
      kind: "property",
      signature: "hour: Int",
      params: [],
      returns: "Int",
      doc: message("The hour, 0 to 23."),
    },
    {
      receiver: "FreightClass",
      name: "number",
      kind: "property",
      signature: "number: Number",
      params: [],
      returns: "Number",
      doc: message("The class as a number: `Class77_5.number` is `77.5`."),
    },
    {
      receiver: "Shipment",
      name: "requested",
      kind: "method",
      signature: "requested(code: CODE) -> Bool",
      params: [{ name: "code", type: "CODE", default: null }],
      returns: "Bool",
      doc: message("Whether the user put an accessorial with this charge code on the shipment."),
    },
  ],
  tableMethods: [],
  units: [{ unit: "lb", dimension: "Weight", factor: "0.45359237", si: "kg" }],
  keywords: ["charge", "review"],
  contextual: ["on", "freight", "review", "total", "lines", "reviews", "applies"],
  schema: {
    roots: [
      {
        name: "lane",
        doc: message("The lane."),
        type: "Lane",
        fields: [
          {
            name: "category",
            type: "LocationCategory?",
            descriptor: {
              kind: "optional",
              inner: {
                kind: "enum",
                name: "LocationCategory",
                variants: ["Terminal", "Warehouse"],
                ordered: false,
              },
            },
            words: message("a location category that may be empty"),
            doc: message("What the location is."),
            fields: [],
          },
          {
            name: "minimumClass",
            type: "FreightClass",
            descriptor: {
              kind: "enum",
              name: "FreightClass",
              variants: ["Class50", "Class55"],
              ordered: true,
            },
            words: message("a freight class"),
            doc: null,
            fields: [],
          },
        ],
      },
      {
        name: "tables",
        doc: message("The organisation's active rate matrices, by code."),
        type: "tables",
        fields: [],
      },
      {
        name: "bill",
        doc: message("The bill so far."),
        type: "Bill",
        fields: [
          {
            name: "freight",
            type: "Money",
            descriptor: { kind: "money" },
            words: message("money"),
            doc: message("The sum of the freight (uncoded) lines billed so far, exact."),
            fields: [],
          },
          {
            name: "accessorials",
            type: "Money",
            descriptor: { kind: "money" },
            words: message("money"),
            doc: message("The sum of the accessorial (coded) lines billed so far, exact."),
            fields: [],
          },
          {
            name: "total",
            type: "Money",
            descriptor: { kind: "money" },
            words: message("money"),
            doc: message("The sum of every line billed so far, exact."),
            fields: [],
          },
        ],
        methods: [
          {
            name: "charge",
            signature: "charge(line: CODE | Text) -> Money?",
            doc: message("The sum of the lines billed so far with this label or charge code."),
          },
        ],
      },
    ],
    records: [],
    enums: [
      {
        name: "LocationCategory",
        variants: ["Terminal", "Warehouse", "DistributionCenter"],
        ordered: false,
      },
      { name: "FacilityType", variants: ["CrossDock", "ColdStorage"], ordered: false },
      {
        name: "Weekday",
        variants: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
        ordered: true,
      },
    ],
  },
};

const fixtureRules = { call: () => CATALOG } as unknown as RulesRuntime;

describe("the reference from the catalog", () => {
  const catalog = tariffCatalog(fixtureRules);

  it("lists the bill so far with its fields and bill.charge", () => {
    const bill = dataEntries(catalog, null, translate).filter((entry) => entry.group === "bill");
    expect(bill.map((entry) => entry.label)).toEqual([
      "bill",
      "bill.freight",
      "bill.accessorials",
      "bill.total",
      "bill.charge(line: CODE | Text) -> Money?",
    ]);
    expect(bill.at(-1)).toMatchObject({
      insert: "bill.charge()",
      cursor: "bill.charge(".length,
      doc: "The sum of the lines billed so far with this label or charge code.",
    });
  });

  it("lists the schema's enums, saying which are ordered", () => {
    const enums = languageEntries(catalog, translate).filter((entry) => entry.group === "Enums");
    expect(enums.map((entry) => [entry.label, entry.meta, entry.doc !== ""])).toEqual([
      ["LocationCategory", "Terminal | Warehouse | DistributionCenter", false],
      ["FacilityType", "CrossDock | ColdStorage", false],
      ["Weekday", "Mon | Tue | Wed | Thu | Fri | Sat | Sun", true],
    ]);
  });

  it("lists the methods on times, freight classes and the shipment, and the Time type", () => {
    const entries = languageEntries(catalog, translate);
    expect(entries.map((entry) => `${entry.group}: ${entry.label}`)).toEqual(
      expect.arrayContaining([
        "Types: Time",
        "On Time: .hour",
        "On FreightClass: .number",
        "On Shipment: .requested(code: CODE) -> Bool",
      ]),
    );
  });

  it("lists std.tiers with the other modules", () => {
    expect(libraryEntries(catalog, null, [], translate)).toEqual([
      expect.objectContaining({ group: "std.tiers", label: "byDistance", insert: "byDistance()" }),
    ]);
  });
});

const context = {
  ...EMPTY_TARIFF_CONTEXT,
  libraries: [
    { path: "org.accessorials", version: "flv_1", source: 'library "org.accessorials"\n' },
    { path: "org.zones", version: "flv_2", source: 'library "org.zones"\n' },
  ],
};

describe("libraryEntries", () => {
  it("lists the organisation's libraries first, each with its approved version and description", () => {
    const entries = libraryEntries(
      tariffCatalog(rulesSync()),
      context,
      [
        {
          path: "org.accessorials",
          libraryId: "fl_1",
          name: "accessorials",
          description: "Fees our tariffs share",
          versionNumber: 3,
        },
      ],
      translate,
    );
    expect(entries.slice(0, 2)).toEqual([
      {
        key: "org:org.accessorials",
        group: "Your libraries",
        label: "org.accessorials",
        meta: "approved v3",
        doc: "Fees our tariffs share",
        insert: "use org.accessorials\n",
      },
      {
        key: "org:org.zones",
        group: "Your libraries",
        label: "org.zones",
        meta: undefined,
        doc: "Import it with use org.zones",
        insert: "use org.zones\n",
      },
    ]);
  });

  it("lists every standard library declaration after them, grouped by module", () => {
    const entries = libraryEntries(tariffCatalog(rulesSync()), null, [], translate);
    expect(entries.some((entry) => entry.group === "std.time" && entry.label === "weekend")).toBe(
      true,
    );
    expect(entries.every((entry) => !entry.key.startsWith("org:"))).toBe(true);
  });
});

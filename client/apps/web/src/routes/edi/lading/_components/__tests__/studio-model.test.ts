import type { LadingCheck, LadingMap, LadingSetting } from "@trenova/shared/edi/lading";
import type { LadingVersion } from "@trenova/shared/types/lading";
import { describe, expect, it } from "vitest";
import {
  applyFileEdits,
  byFile,
  documentStem,
  entryPath,
  freeName,
  isDirty,
  isEditable,
  kebabName,
  ladingName,
  lineOf,
  newPathProblem,
  pickVersion,
  projectMap,
  problemCounts,
  projectProblems,
  releaseStages,
  renameEntry,
  runValues,
  sampleFits,
  sampleFormat,
  sampleInput,
  samplesFor,
  starterSamplePath,
  starterSource,
  settingDraft,
  settingUnit,
  settingValue,
} from "../studio-model";

function version(id: string, status: LadingVersion["status"], number: number): LadingVersion {
  return {
    id,
    projectId: "ldp_1",
    number,
    status,
    files: [],
    samples: [],
    manifest: {} as LadingCheck,
    errorCount: 0,
    warningCount: 0,
    version: 1,
    createdAt: 1_760_000_000,
    updatedAt: 1_760_000_000,
  };
}

const versions = [
  version("ldv_3", "Draft", 3),
  version("ldv_2", "Active", 2),
  version("ldv_1", "Superseded", 1),
];

describe("pickVersion", () => {
  it("opens the version the address names when the project has it", () => {
    expect(pickVersion(versions, "ldv_1", "ldv_2")?.id).toBe("ldv_1");
  });

  it("opens the draft, else the active version, else the newest", () => {
    expect(pickVersion(versions, "ldv_gone", "ldv_2")?.id).toBe("ldv_3");
    expect(pickVersion(versions.slice(1), null, "ldv_2")?.id).toBe("ldv_2");
    expect(pickVersion([version("ldv_9", "Archived", 9)], null, null)?.id).toBe("ldv_9");
    expect(pickVersion([], null, null)).toBeNull();
  });
});

describe("isEditable", () => {
  it("writes only to a draft", () => {
    expect(isEditable({ status: "Draft" })).toBe(true);
    expect(isEditable({ status: "Certified" })).toBe(false);
    expect(isEditable(null)).toBe(false);
  });
});

describe("isDirty", () => {
  const saved = {
    files: [{ path: "partner.lading", source: "partner acme" }],
    samples: [{ path: "samples/a.x12", content: "ISA" }],
  };

  it("is clean when every file and sample reads as saved, in any order", () => {
    expect(isDirty(saved, { files: [...saved.files], samples: [...saved.samples] })).toBe(false);
  });

  it("is dirty when a file's text, a sample, or the set of files changed", () => {
    expect(
      isDirty(saved, { ...saved, files: [{ path: "partner.lading", source: "partner acme2" }] }),
    ).toBe(true);
    expect(isDirty(saved, { ...saved, samples: [] })).toBe(true);
    expect(
      isDirty(saved, {
        ...saved,
        files: [...saved.files, { path: "maps/x.lading", source: "map x" }],
      }),
    ).toBe(true);
  });
});

describe("newPathProblem", () => {
  const taken = ["partner.lading", "samples/a.x12"];

  it("takes a new relative path, a source file's ending in .lading", () => {
    expect(newPathProblem("maps/tender.lading", "file", taken)).toBeNull();
    expect(newPathProblem("samples/b.x12", "sample", taken)).toBeNull();
  });

  it("refuses a path that is empty, leaves the project, is not Lading or is taken", () => {
    expect(newPathProblem("  ", "file", taken)).toBe("empty");
    expect(newPathProblem("/etc/x.lading", "file", taken)).toBe("notRelative");
    expect(newPathProblem("maps/../x.lading", "file", taken)).toBe("notRelative");
    expect(newPathProblem("maps//x.lading", "file", taken)).toBe("notRelative");
    expect(newPathProblem("maps/tender.txt", "file", taken)).toBe("notLading");
    expect(newPathProblem("samples/a.x12", "sample", taken)).toBe("taken");
  });

  it("takes a sample only by an ending a sample is read by", () => {
    expect(newPathProblem("samples/b.edi", "sample", taken)).toBeNull();
    expect(newPathProblem("samples/b.txt", "sample", taken)).toBeNull();
    expect(newPathProblem("samples/rows.csv", "sample", taken)).toBeNull();
    expect(newPathProblem("samples/tender.document.json", "sample", taken)).toBeNull();
    expect(newPathProblem("samples/b.lading", "sample", taken)).toBe("notSample");
    expect(newPathProblem("samples/b", "sample", taken)).toBe("notSample");
    expect(newPathProblem("samples/b.pdf", "sample", taken)).toBe("notSample");
  });
});

describe("sampleFormat", () => {
  it("reads .x12, .edi and .txt as X12, .document.json as a document, the rest as named", () => {
    expect(sampleFormat("samples/a.x12")).toBe("x12");
    expect(sampleFormat("samples/a.EDI")).toBe("x12");
    expect(sampleFormat("samples/a.txt")).toBe("x12");
    expect(sampleFormat("samples/status.document.json")).toBe("document");
    expect(sampleFormat("samples/order.json")).toBe("json");
    expect(sampleFormat("samples/orders.csv")).toBe("csv");
    expect(sampleFormat("samples/notes.md")).toBeNull();
  });
});

describe("entryPath", () => {
  it("puts each kind in its folder with its ending, without repeating a typed ending", () => {
    expect(entryPath("map", "tender-in")).toBe("maps/tender-in.lading");
    expect(entryPath("map", "tender-in.lading")).toBe("maps/tender-in.lading");
    expect(entryPath("rules", "tender-checks")).toBe("rules/tender-checks.lading");
    expect(entryPath("flow", "tender")).toBe("flows/tender.lading");
    expect(entryPath("guide", "acme-204")).toBe("guides/acme-204.lading");
    expect(entryPath("library", "common")).toBe("lib/common.lading");
    expect(entryPath("partner", "anything")).toBe("partner.lading");
    expect(entryPath("x12", "tender-original")).toBe("samples/tender-original.x12");
    expect(entryPath("csv", "orders.csv")).toBe("samples/orders.csv");
    expect(entryPath("json", "order")).toBe("samples/order.json");
    expect(entryPath("document", "status")).toBe("samples/status.document.json");
  });

  it("keeps a subfolder typed in the name, and is empty for no name", () => {
    expect(entryPath("map", "acme/tender-in")).toBe("maps/acme/tender-in.lading");
    expect(entryPath("map", "   ")).toBe("");
  });
});

describe("freeName", () => {
  it("keeps a free name and numbers a taken one with the first free number", () => {
    const taken = ["maps/tender-in.lading", "maps/tender-in-2.lading", "samples/a.x12"];
    expect(freeName("map", "status-in", taken)).toBe("status-in");
    expect(freeName("map", "tender-in", taken)).toBe("tender-in-3");
    expect(freeName("x12", "a", taken)).toBe("a-2");
  });
});

describe("names", () => {
  it("makes Lading and file names from either form", () => {
    expect(ladingName("tender-in")).toBe("tenderIn");
    expect(ladingName("Acme Manufacturing")).toBe("acmeManufacturing");
    expect(kebabName("tenderIn")).toBe("tender-in");
    expect(kebabName("acmeManufacturing")).toBe("acme-manufacturing");
  });

  it("names a document's files by its last word", () => {
    expect(documentStem("loadTender")).toBe("tender");
    expect(documentStem("tenderResponse")).toBe("response");
    expect(documentStem("shipmentStatus")).toBe("status");
    expect(documentStem("freightInvoice")).toBe("invoice");
  });
});

describe("starterSource", () => {
  it("starts an inbound X12 map from the guide it reads to the document it makes", () => {
    expect(
      starterSource("maps/tender-in.lading", {
        kind: "map",
        direction: "inbound",
        input: "x12",
        guide: "acme.204",
        document: "loadTender",
      }),
    ).toBe("map tenderIn\n  from acme.204\n  to   loadTender\n");
  });

  it("starts an outbound map from the document to the guide it writes with", () => {
    expect(
      starterSource("maps/status-out.lading", {
        kind: "map",
        direction: "outbound",
        document: "shipmentStatus",
        guide: "x12.004010.214",
      }),
    ).toBe("map statusOut\n  from shipmentStatus\n  to   x12.004010.214\n");
  });

  it("starts a CSV map with one column and a JSON map from json", () => {
    expect(
      starterSource("maps/orders-csv.lading", {
        kind: "map",
        direction: "inbound",
        input: "csv",
        document: "loadTender",
      }),
    ).toBe("map ordersCsv\n  from csv {\n    reference: Text\n  }\n  to   loadTender\n");
    expect(
      starterSource("maps/order-json.lading", {
        kind: "map",
        direction: "inbound",
        input: "json",
        document: "loadTender",
      }),
    ).toBe("map orderJson\n  from json\n  to   loadTender\n");
  });

  it("names a guide with dots from its file and quotes its title safely", () => {
    expect(
      starterSource("guides/acme-204.lading", {
        kind: "guide",
        base: "x12.004010.204",
        title: "ACME Load Tender Guide rev 3.2",
      }),
    ).toBe('guide acme.204 from x12.004010.204 "ACME Load Tender Guide rev 3.2"\n');
    expect(
      starterSource("guides/Acme_214_Fleet.lading", {
        kind: "guide",
        base: "acme.214",
        title: 'The "fleet" {rev}',
      }),
    ).toBe("guide acme.214.fleet from acme.214 \"The 'fleet' rev\"\n");
  });

  it("starts rules for the map they check, a flow with the partner, and a library", () => {
    expect(starterSource("rules/tender-checks.lading", { kind: "rules", map: "tenderIn" })).toBe(
      "rules tenderChecks for tenderIn\n",
    );
    expect(starterSource("flows/tender.lading", { kind: "flow", partner: "acme" })).toBe(
      "flow tender with acme\n",
    );
    expect(starterSource("lib/204-helpers.lading", { kind: "library" })).toBe(
      "library file204Helpers\n",
    );
  });
});

describe("renameEntry", () => {
  const project = {
    files: [
      { path: "maps/tender-in.lading", source: "map tenderIn" },
      {
        path: "flows/tender.lading",
        source:
          'test "a" {\n  given file "samples/a.x12"\n}\ntest "b" { given file "samples/a.x12.bak" }',
      },
    ],
    samples: [
      { path: "samples/a.x12", content: "ISA" },
      { path: "samples/b.x12", content: "ISA*B" },
    ],
  };

  it("moves a sample and every test that reads it by its quoted path, and nothing else", () => {
    const moved = renameEntry(project, "samples/a.x12", "samples/original.x12");
    expect(moved.samples).toEqual([
      { path: "samples/original.x12", content: "ISA" },
      { path: "samples/b.x12", content: "ISA*B" },
    ]);
    expect(moved.files[1]?.source).toBe(
      'test "a" {\n  given file "samples/original.x12"\n}\ntest "b" { given file "samples/a.x12.bak" }',
    );
    expect(moved.files[0]).toBe(project.files[0]);
  });

  it("moves a source file without touching any file's text", () => {
    const moved = renameEntry(project, "maps/tender-in.lading", "maps/acme/tender-in.lading");
    expect(moved.files).toEqual([
      { path: "maps/acme/tender-in.lading", source: "map tenderIn" },
      project.files[1],
    ]);
    expect(moved.samples).toEqual(project.samples);
  });
});

describe("projectMap", () => {
  const checked = (path: string, kind: string) => ({
    path,
    kind,
    name: null,
    outline: [],
    folds: [],
    diagnostics: [],
  });
  const map = (
    name: string,
    path: string,
    direction: "inbound" | "outbound",
    set: string | null,
    source: LadingMap["source"],
  ): LadingMap => ({
    name,
    path,
    direction,
    from: "",
    to: "",
    document: "loadTender",
    set,
    source,
  });
  const paths = [
    ["partner.lading", "partner"],
    ["flows/tender.lading", "flow"],
    ["flows/backhaul.lading", "flow"],
    ["maps/tender-in.lading", "map"],
    ["maps/response-out.lading", "map"],
    ["maps/status-out.lading", "map"],
    ["maps/orders-csv.lading", "map"],
    ["rules/tender-checks.lading", "rules"],
    ["rules/extra-checks.lading", "rules"],
    ["rules/orders-checks.lading", "rules"],
    ["lib/common.lading", "library"],
    ["guides/acme-204.lading", "guide"],
    ["scratch.lading", ""],
  ] as const;
  const files = [
    ...paths.map(([path]) => ({ path, source: "" })),
    { path: "new.lading", source: "" },
  ];
  const samples = [
    { path: "samples/tender.x12", content: "" },
    { path: "samples/orders.csv", content: "" },
    { path: "samples/status.document.json", content: "" },
  ];
  const project: LadingCheck = {
    files: paths.map(([path, kind]) => checked(path, kind)),
    partner: null,
    maps: [
      map("tenderIn", "maps/tender-in.lading", "inbound", "204", "x12"),
      map("responseOut", "maps/response-out.lading", "outbound", "990", "document"),
      map("statusOut", "maps/status-out.lading", "outbound", "214", "document"),
      map("ordersCsv", "maps/orders-csv.lading", "inbound", null, "csv"),
    ],
    flows: [
      {
        name: "tender",
        path: "flows/tender.lading",
        clauses: [
          { kind: "receive", set: "204", map: "tenderIn", rules: "tenderChecks" },
          { kind: "respond", set: "990", map: "responseOut" },
        ],
      },
      {
        name: "backhaul",
        path: "flows/backhaul.lading",
        clauses: [
          { kind: "send", set: "214", map: "statusOut" },
          { kind: "receive", set: "204", map: "tenderIn" },
          { kind: "expect", set: "997", map: null },
        ],
      },
    ],
    rules: [
      { name: "tenderChecks", path: "rules/tender-checks.lading", map: "tenderIn" },
      { name: "extraChecks", path: "rules/extra-checks.lading", map: "tenderIn" },
      { name: "ordersChecks", path: "rules/orders-checks.lading", map: "ordersCsv" },
    ],
    libraries: ["common"],
    guides: [
      {
        name: "acme.204",
        path: "guides/acme-204.lading",
        set: "204",
        version: "004010",
        base: "x12.004010.204",
      },
      { name: "x12.004010.204", path: null, set: "204", version: "004010", base: null },
    ],
    errors: 0,
    warnings: 0,
  };
  const pathsOf = (entries: readonly { path: string }[]) => entries.map((entry) => entry.path);

  it("puts under each flow the maps and rules its clauses use, in clause order", () => {
    const tree = projectMap(files, samples, project);
    expect(tree.flows.map((branch) => [branch.flow.path, pathsOf(branch.children)])).toEqual([
      [
        "flows/backhaul.lading",
        [
          "maps/status-out.lading",
          "maps/tender-in.lading",
          "rules/tender-checks.lading",
          "rules/extra-checks.lading",
        ],
      ],
      [
        "flows/tender.lading",
        [
          "maps/tender-in.lading",
          "rules/tender-checks.lading",
          "rules/extra-checks.lading",
          "maps/response-out.lading",
        ],
      ],
    ]);
  });

  it("keeps the partner, the shared guides then libraries, the files no flow reaches, and the samples", () => {
    const tree = projectMap(files, samples, project);
    expect(pathsOf(tree.partner)).toEqual(["partner.lading"]);
    expect(pathsOf(tree.shared)).toEqual(["guides/acme-204.lading", "lib/common.lading"]);
    expect(pathsOf(tree.loose)).toEqual([
      "maps/orders-csv.lading",
      "new.lading",
      "rules/orders-checks.lading",
      "scratch.lading",
    ]);
    expect(pathsOf(tree.samples)).toEqual([
      "samples/orders.csv",
      "samples/status.document.json",
      "samples/tender.x12",
    ]);
  });

  it("labels each entry without its kind's folder and carries what the check knows of it", () => {
    const tree = projectMap(files, samples, project);
    const entries = [
      ...tree.partner,
      ...tree.flows.flatMap((branch) => [branch.flow, ...branch.children]),
      ...tree.shared,
      ...tree.loose,
      ...tree.samples,
    ];
    const at = (path: string) => entries.find((entry) => entry.path === path);
    expect(at("maps/tender-in.lading")).toMatchObject({
      kind: "map",
      label: "tender-in",
      map: { direction: "inbound", set: "204" },
    });
    expect(at("rules/extra-checks.lading")).toMatchObject({
      kind: "rules",
      label: "extra-checks",
      rulesFor: "tenderIn",
    });
    expect(at("guides/acme-204.lading")).toMatchObject({
      kind: "guide",
      label: "acme-204",
      guideBase: "x12.004010.204",
    });
    expect(at("scratch.lading")).toMatchObject({ kind: "other", label: "scratch" });
    expect(at("new.lading")).toMatchObject({ kind: "other", label: "new" });
    expect(at("samples/status.document.json")).toMatchObject({
      kind: "sample",
      label: "status.document.json",
      format: "document",
    });
  });

  it("is empty for an empty project", () => {
    expect(
      projectMap([], [], { ...project, files: [], maps: [], flows: [], rules: [], guides: [] }),
    ).toEqual({ partner: [], flows: [], shared: [], loose: [], samples: [] });
  });
});

const check: LadingCheck = {
  files: [
    {
      path: "maps/b.lading",
      kind: "map",
      name: "b",
      outline: [],
      folds: [],
      diagnostics: [
        {
          range: { from: 40, to: 44 },
          severity: "warning",
          code: "unused",
          message: { message: "Unused" },
          fixes: [],
          related: [],
        },
      ],
    },
    {
      path: "maps/a.lading",
      kind: "map",
      name: "a",
      outline: [],
      folds: [],
      diagnostics: [
        {
          range: { from: 9, to: 12 },
          severity: "error",
          code: "type-mismatch",
          message: { message: "Mismatch" },
          fixes: [],
          related: [],
        },
        {
          range: { from: 1, to: 2 },
          severity: "warning",
          code: "unused",
          message: { message: "Unused" },
          fixes: [],
          related: [],
        },
      ],
    },
  ],
  partner: null,
  maps: [],
  flows: [],
  rules: [],
  libraries: [],
  guides: [],
  errors: 1,
  warnings: 2,
};

describe("byFile", () => {
  it("puts each entry under its file, files in the order of their first entry", () => {
    const entries = [
      { path: "maps/b.lading", id: 1 },
      { path: "maps/a.lading", id: 2 },
      { path: "maps/b.lading", id: 3 },
    ];
    expect(byFile(entries)).toEqual([
      { path: "maps/b.lading", entries: [entries[0], entries[2]] },
      { path: "maps/a.lading", entries: [entries[1]] },
    ]);
    expect(byFile([])).toEqual([]);
  });
});

describe("projectProblems", () => {
  it("lists errors first, then by file and position", () => {
    expect(projectProblems(check).map((problem) => [problem.path, problem.code])).toEqual([
      ["maps/a.lading", "type-mismatch"],
      ["maps/a.lading", "unused"],
      ["maps/b.lading", "unused"],
    ]);
  });

  it("counts each file's errors and warnings", () => {
    expect(problemCounts(check).get("maps/a.lading")).toEqual({ errors: 1, warnings: 1 });
    expect(problemCounts(check).get("maps/b.lading")).toEqual({ errors: 0, warnings: 1 });
  });
});

describe("sampleInput", () => {
  it("reads a sample by its extension", () => {
    expect(sampleInput({ path: "samples/a.x12", content: "ISA*" })).toEqual({ x12: "ISA*" });
    expect(sampleInput({ path: "samples/orders.CSV", content: "a,b" })).toEqual({ csv: "a,b" });
    expect(sampleInput({ path: "samples/o.json", content: '{"a":1}' })).toEqual({
      json: { a: 1 },
    });
  });

  it("gives a Trenova document sample to an outbound map as a document, not as JSON", () => {
    expect(
      sampleInput({ path: "samples/invoice.document.json", content: '{"number":"INV-1"}' }),
    ).toEqual({ document: { number: "INV-1" } });
    expect(sampleInput({ path: "samples/broken.document.json", content: "{" })).toEqual({
      document: "{",
    });
  });
});

describe("lineOf", () => {
  it("counts the lines before an offset", () => {
    expect(lineOf("a\nb\nc", 0)).toBe(1);
    expect(lineOf("a\nb\nc", 4)).toBe(3);
    expect(lineOf("a", 99)).toBe(1);
  });
});

describe("runValues", () => {
  it("lists each read and written value with the elements it came from", () => {
    expect(
      runValues({
        transactions: [
          {
            index: 0,
            status: "accepted",
            findings: [],
            document: null,
            outcomes: [],
            fixes: [],
            unread: [],
            lineage: [
              {
                path: "loadTender.shipmentId",
                value: "SH1",
                sources: [{ path: "B2.04", segment: "B2", element: 4 }],
                steps: ["trim"],
              },
            ],
          },
        ],
        output: {
          text: "",
          transaction: "990",
          document: null,
          findings: [],
          validation: [],
          outcomes: [],
          lineage: [{ path: "B1.02", value: "SH1", sources: [{ path: "shipmentId" }], steps: [] }],
        },
        needs: [],
        unmapped: [],
        steps: 4,
      }),
    ).toEqual([
      { path: "loadTender.shipmentId", value: "SH1", sources: ["B204"], steps: ["trim"] },
      { path: "B1.02", value: "SH1", sources: ["shipmentId"], steps: [] },
    ]);
  });

  it("has nothing to show for a run that failed before reading", () => {
    expect(
      runValues({
        transactions: [],
        needs: [],
        unmapped: [],
        error: { message: "No map {0}", args: ["x"] },
        steps: 0,
      }),
    ).toEqual([]);
  });
});

function setting(type: string, defaultValue: string | null): LadingSetting {
  return { name: "s", type, default: defaultValue, range: null, doc: null, value: null };
}

describe("settings", () => {
  it("starts each field from the stored value, else blank so the default holds", () => {
    expect(settingDraft(setting("Bool", "false"), undefined)).toBe(false);
    expect(settingDraft(setting("Bool", "false"), true)).toBe(true);
    expect(settingDraft(setting("Int", "3"), undefined)).toBe("");
    expect(settingDraft(setting("Int", "3"), 5)).toBe("5");
    expect(settingDraft(setting("Weight", "45_000 lb"), { value: 40000, unit: "lb" })).toBe(
      "40000",
    );
    expect(settingDraft(setting("Money", "10 USD"), { amount: "12.50" })).toBe("12.50");
  });

  it("enters a measure in the stored value's unit, else the default's", () => {
    expect(settingUnit(setting("Weight", "45_000 lb"), undefined)).toBe("lb");
    expect(settingUnit(setting("Weight", "45_000 lb"), { value: 20, unit: "kg" })).toBe("kg");
    expect(settingUnit(setting("Weight", null), undefined)).toBe("");
  });

  it("saves each type in the shape the server takes, null for the default", () => {
    expect(settingValue(setting("Bool", "false"), true, "")).toBe(true);
    expect(settingValue(setting("Int", "3"), "1_000", "")).toBe(1000);
    expect(settingValue(setting("Duration", "60"), "", "")).toBeNull();
    expect(settingValue(setting("Decimal", "1.5"), "2.25", "")).toBe("2.25");
    expect(settingValue(setting("Text", '""'), " ACME ", "")).toBe(" ACME ");
    expect(settingValue(setting("Weight", "45_000 lb"), "40000", "lb")).toEqual({
      value: 40000,
      unit: "lb",
    });
  });

  it("refuses text that is not a value of the setting's type", () => {
    expect(settingValue(setting("Int", "3"), "2.5", "")).toBeUndefined();
    expect(settingValue(setting("Percent", "5"), "five", "")).toBeUndefined();
    expect(settingValue(setting("Weight", null), "10", "")).toBeUndefined();
  });
});

describe("applyFileEdits", () => {
  const files = [
    { path: "maps/a.lading", source: "let x = 1\nlet y = x\n" },
    { path: "maps/b.lading", source: "use x\n" },
  ];

  it("applies several edits to one file against its original offsets", () => {
    expect(
      applyFileEdits(files, [
        { path: "maps/a.lading", range: { from: 4, to: 5 }, insert: "count" },
        { path: "maps/a.lading", range: { from: 18, to: 19 }, insert: "count" },
      ])[0],
    ).toEqual({ path: "maps/a.lading", source: "let count = 1\nlet y = count\n" });
  });

  it("edits other files and leaves untouched ones as they were", () => {
    const next = applyFileEdits(files, [
      { path: "maps/b.lading", range: { from: 4, to: 5 }, insert: "count" },
    ]);
    expect(next[0]).toBe(files[0]);
    expect(next[1]).toEqual({ path: "maps/b.lading", source: "use count\n" });
  });

  it("creates the file an edit names when the project lacks it", () => {
    expect(
      applyFileEdits(files, [
        { path: "lib/common.lading", range: { from: 0, to: 0 }, insert: "library common\n" },
      ]).at(-1),
    ).toEqual({ path: "lib/common.lading", source: "library common\n" });
  });
});

describe("releaseStages", () => {
  const summary = (newlyRejected: number) => ({
    total: 10,
    same: 10 - newlyRejected,
    differ: newlyRejected,
    newlyRejected,
    newlyHeld: 0,
    newlyDeclined: 0,
    failed: 0,
  });
  const draft = { id: "ldv_2", status: "Draft" as const, shadowUntil: null };
  const states = (stages: ReturnType<typeof releaseStages>) =>
    stages.map((stage) => [stage.key, stage.state]);

  it("counts an inbound map replayed once a finished replay of it has no new rejection", () => {
    const stages = releaseStages({
      version: draft,
      activeVersionId: "ldv_1",
      dirty: false,
      inboundMaps: ["tenderIn", "statusIn", "invoiceIn"],
      replays: [
        { map: "tenderIn", status: "Completed", summary: summary(2) },
        { map: "tenderIn", status: "Completed", summary: summary(0) },
        { map: "statusIn", status: "Running", summary: null },
        { map: "invoiceIn", status: "Failed", summary: null },
      ],
    });
    expect(stages[1]).toEqual({ key: "replayed", state: "running", clean: 1, total: 3 });
    expect(states(stages)).toEqual([
      ["saved", "done"],
      ["replayed", "running"],
      ["certified", "pending"],
      ["shadow", "pending"],
      ["live", "pending"],
    ]);
  });

  it("fails the replay step when a map's every finished replay newly rejects", () => {
    const stages = releaseStages({
      version: draft,
      activeVersionId: "ldv_1",
      dirty: true,
      inboundMaps: ["tenderIn", "statusIn"],
      replays: [{ map: "tenderIn", status: "Completed", summary: summary(1) }],
    });
    expect(states(stages).slice(0, 2)).toEqual([
      ["saved", "attention"],
      ["replayed", "failed"],
    ]);
  });

  it("skips replaying with no active version, for the active version, or with no inbound map", () => {
    const base = { version: draft, dirty: false, replays: [] };
    expect(releaseStages({ ...base, activeVersionId: null, inboundMaps: ["tenderIn"] })[1]).toEqual(
      { key: "replayed", state: "skipped", clean: 0, total: 1, reason: "noActive" },
    );
    expect(
      releaseStages({ ...base, activeVersionId: "ldv_2", inboundMaps: ["tenderIn"] })[1],
    ).toMatchObject({ state: "skipped", reason: "isActive" });
    expect(releaseStages({ ...base, activeVersionId: "ldv_1", inboundMaps: [] })[1]).toMatchObject({
      state: "skipped",
      reason: "noInbound",
    });
  });

  it("follows a version from certified through shadow to live and superseded", () => {
    const at = (status: LadingVersion["status"], shadowUntil: number | null) =>
      states(
        releaseStages({
          version: { id: "ldv_2", status, shadowUntil },
          activeVersionId: "ldv_1",
          dirty: true,
          inboundMaps: [],
          replays: [],
        }),
      ).filter(([key]) => key !== "replayed");
    expect(at("Certified", null)).toEqual([
      ["saved", "done"],
      ["certified", "done"],
      ["shadow", "pending"],
      ["live", "pending"],
    ]);
    expect(at("Shadow", 1_760_000_000)).toEqual([
      ["saved", "done"],
      ["certified", "done"],
      ["shadow", "running"],
      ["live", "pending"],
    ]);
    expect(at("Active", null)).toEqual([
      ["saved", "done"],
      ["certified", "done"],
      ["shadow", "skipped"],
      ["live", "done"],
    ]);
    expect(at("Superseded", 1_760_000_000)).toEqual([
      ["saved", "done"],
      ["certified", "done"],
      ["shadow", "done"],
      ["live", "done"],
    ]);
  });
});

describe("samplesFor", () => {
  const map = (source: LadingMap["source"], set: string | null): LadingMap => ({
    name: "tenderIn",
    path: "maps/tender-in.lading",
    direction: source === "document" ? "outbound" : "inbound",
    from: "acme.204",
    to: "loadTender",
    document: "loadTender",
    set,
    source,
  });
  const tender = { path: "samples/tender.x12", content: "ST*204*0001~B2**TRNV~SE*3*0001~" };
  const invoice = { path: "samples/invoice.x12", content: "ST*210*0001~B3**INV-1~SE*3*0001~" };
  const orders = { path: "samples/orders.csv", content: "order,pickup" };
  const order = { path: "samples/order.json", content: "{}" };
  const document = { path: "samples/invoice.document.json", content: "{}" };
  const samples = [invoice, orders, tender, order, document];

  it("puts first the X12 samples holding the map's set, the rest after, in project order", () => {
    expect(samplesFor(map("x12", "204"), samples)).toEqual({
      fitting: [tender],
      others: [invoice, orders, order, document],
    });
  });

  it("matches a CSV, JSON or outbound map by the sample's format", () => {
    expect(samplesFor(map("csv", null), samples).fitting).toEqual([orders]);
    expect(samplesFor(map("json", null), samples).fitting).toEqual([order]);
    expect(samplesFor(map("document", "210"), samples).fitting).toEqual([document]);
  });

  it("keeps every sample among the others when no map is picked", () => {
    expect(samplesFor(undefined, samples)).toEqual({ fitting: [], others: samples });
  });

  it("says a sample holding another set does not fit", () => {
    expect(sampleFits(map("x12", "204"), invoice)).toBe(false);
    expect(sampleFits(map("x12", "210"), invoice)).toBe(true);
  });
});

describe("starterSamplePath", () => {
  it("names a new map's sample after its file, in the sample's format, and never takes a path", () => {
    expect(starterSamplePath("maps/invoice-in.lading", "x12", [])).toBe("samples/invoice-in.x12");
    expect(starterSamplePath("maps/invoice-out.lading", "document", [])).toBe(
      "samples/invoice-out.document.json",
    );
    expect(
      starterSamplePath("maps/orders.lading", "csv", [
        "samples/orders.csv",
        "samples/orders-2.csv",
      ]),
    ).toBe("samples/orders-3.csv");
  });
});

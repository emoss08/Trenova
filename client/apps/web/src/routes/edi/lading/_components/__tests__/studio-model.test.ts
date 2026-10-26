import type { LadingCheck, LadingSetting } from "@trenova/shared/edi/lading";
import type { LadingVersion } from "@trenova/shared/types/lading";
import { describe, expect, it } from "vitest";
import {
  applyFileEdits,
  isDirty,
  isEditable,
  lineOf,
  newFileSource,
  newPathProblem,
  pickVersion,
  problemCounts,
  projectProblems,
  runValues,
  sampleInput,
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
});

describe("newFileSource", () => {
  it("starts a file with the header its folder suggests", () => {
    expect(newFileSource("maps/status-in.lading", "acme")).toBe(
      "map status_in\n  from x12.004010.204\n  to   loadTender\n",
    );
    expect(newFileSource("flows/tender.lading", "acme")).toBe("flow tender with acme\n");
    expect(newFileSource("common.lading", "acme")).toBe("library common\n");
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

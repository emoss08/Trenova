import { translate } from "@trenova/shared/i18n/runtime";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffCatalog } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { libraryEntries } from "../reference-pane";

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

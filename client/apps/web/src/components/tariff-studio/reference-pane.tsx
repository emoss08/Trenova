import { useActiveEditorInsert } from "@/components/formula-editor/active-editor";
import { tariffDocText } from "@/components/formula-editor/tariff-doc";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import { PlusIcon, SearchLgIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  tariffCatalog,
  type TariffCatalog,
  type TariffCatalogField,
} from "@trenova/shared/rules/tariff";
import type { TariffContext } from "@trenova/shared/types/tariff";
import { useMemo, useState } from "react";
import { REFERENCE_SEARCH_ID } from "./reference-search";

const ROW_HEIGHT = 44;

/** One thing the reference lists: what is inserted, how it reads, and its doc. */
export type ReferenceEntry = {
  key: string;
  group: string;
  label: string;
  meta?: string;
  doc: string;
  insert: string;
  /** Where the cursor lands, counted from the start of the insertion. */
  cursor?: number;
};

export type ReferenceTab = "data" | "library" | "language";

function fieldEntries(
  path: string,
  fields: readonly TariffCatalogField[],
  group: string,
): ReferenceEntry[] {
  return fields.flatMap((field) => {
    const name = `${path}.${field.name}`;
    const own: ReferenceEntry = {
      key: `field:${name}`,
      group,
      label: name,
      meta: field.type,
      doc: tariffDocText(field.doc) ?? "",
      insert: name,
    };
    return [own, ...fieldEntries(name, field.fields, group)];
  });
}

/** Everything the data tab lists: the schema's roots and fields, the tables and the codes. */
export function dataEntries(
  catalog: TariffCatalog,
  context: TariffContext | null,
  t: TranslateFn,
): ReferenceEntry[] {
  const roots = catalog.schema.roots.filter((root) => root.name !== "tables");
  const schema = roots.flatMap((root): ReferenceEntry[] => [
    {
      key: `root:${root.name}`,
      group: root.name,
      label: root.name,
      meta: root.type,
      doc: tariffDocText(root.doc) ?? "",
      insert: root.name,
    },
    ...fieldEntries(root.name, root.fields, root.name),
  ]);
  const tables = (context?.tables ?? []).map(
    (table): ReferenceEntry => ({
      key: `table:${table.code}`,
      group: t("Rate tables"),
      label: `tables.${table.code}`,
      meta: t("{0} → {1}", table.axes.map((axis) => axis.label || axis.kind).join(", "), table.valueType),
      doc: table.name,
      insert: `tables.${table.code}[]`,
      cursor: `tables.${table.code}[`.length,
    }),
  );
  const codes = (context?.codes ?? [])
    .filter((code) => code.active)
    .map(
      (code): ReferenceEntry => ({
        key: `code:${code.code}`,
        group: t("Accessorial charge codes"),
        label: code.code,
        meta: t("charge … as {0}", code.code),
        doc: code.description,
        insert: code.code,
      }),
    );
  return [...schema, ...tables, ...codes];
}

/** The standard library and the organisation's libraries. */
export function libraryEntries(
  catalog: TariffCatalog,
  context: TariffContext | null,
  t: TranslateFn,
): ReferenceEntry[] {
  const std = catalog.modules.flatMap((module) =>
    module.declarations.map(
      (declaration): ReferenceEntry => ({
        key: `std:${module.path}.${declaration.name}`,
        group: module.path,
        label: declaration.name,
        meta: declaration.signature,
        doc: tariffDocText(declaration.doc) ?? "",
        insert: declaration.kind === "fn" ? `${declaration.name}()` : declaration.name,
        cursor: declaration.kind === "fn" ? declaration.name.length + 1 : undefined,
      }),
    ),
  );
  const org = (context?.libraries ?? []).map(
    (library): ReferenceEntry => ({
      key: `org:${library.path}`,
      group: t("Your libraries"),
      label: library.path,
      meta: library.version ? t("pinned to {0}", library.version) : undefined,
      doc: t("Import it with use {0}", library.path),
      insert: `use ${library.path}\n`,
    }),
  );
  return [...org, ...std];
}

/** Methods and properties by the type they are on, the built-in types, units and keywords. */
export function languageEntries(catalog: TariffCatalog, t: TranslateFn): ReferenceEntry[] {
  const methods = catalog.methods.map(
    (method): ReferenceEntry => ({
      key: `method:${method.receiver}.${method.name}`,
      group: t("On {0}", method.receiver),
      label: method.kind === "property" ? `.${method.name}` : `.${method.signature}`,
      meta: method.returns,
      doc: tariffDocText(method.doc) ?? "",
      insert: method.kind === "property" ? `.${method.name}` : `.${method.name}()`,
      cursor: method.kind === "property" ? undefined : method.name.length + 2,
    }),
  );
  const types = catalog.types.map(
    (type): ReferenceEntry => ({
      key: `type:${type.name}`,
      group: t("Types"),
      label: type.name,
      doc: tariffDocText(type.doc) ?? "",
      insert: type.name,
    }),
  );
  const units = catalog.units.map(
    (unit): ReferenceEntry => ({
      key: `unit:${unit.unit}`,
      group: t("Units"),
      label: unit.unit,
      meta: unit.dimension,
      doc: "",
      insert: unit.unit,
    }),
  );
  return [...types, ...methods, ...units];
}

function ReferenceRow({ entry, onInsert }: { entry: ReferenceEntry; onInsert: () => void }) {
  return (
    <button
      type="button"
      onClick={onInsert}
      className="group ui-focus-ring hover:bg-surface-hover flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left"
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex min-w-0 items-baseline gap-2">
          <span className="truncate font-mono text-xs">{entry.label}</span>
          {entry.meta && (
            <span className="text-foreground-subtle min-w-0 truncate font-mono text-2xs">
              {entry.meta}
            </span>
          )}
        </span>
        {entry.doc && (
          <span className="text-foreground-muted line-clamp-1 text-xs">{entry.doc}</span>
        )}
      </span>
      <PlusIcon className="text-foreground-subtle mt-0.5 size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100" />
    </button>
  );
}

function GroupHeading({ label }: { label: string }) {
  return (
    <div className="bg-card text-foreground-subtle px-2 pt-2 pb-1 font-mono text-xs">{label}</div>
  );
}

function rowsOf(entries: readonly ReferenceEntry[], insert: (entry: ReferenceEntry) => void) {
  const rows: VirtualRow[] = [];
  let group: string | null = null;
  for (const entry of entries) {
    if (entry.group !== group) {
      group = entry.group;
      const label = entry.group;
      rows.push({ key: `heading:${label}`, sticky: true, render: () => <GroupHeading label={label} /> });
    }
    rows.push({
      key: entry.key,
      render: () => <ReferenceRow entry={entry} onInsert={() => insert(entry)} />,
    });
  }
  return rows;
}

/**
 * What a Tariff file can name, from `tariff.catalog` and the organisation's context: the data
 * roots and their fields, the rate tables and accessorial codes; the standard library and the
 * organisation's libraries; the methods by type, the built-in types and the units. `/` searches
 * it, and a row drops its text at the cursor of the editor touched last. The list is
 * virtualized with each group's heading pinned.
 */
export function ReferencePane({ context }: { context: TariffContext | null }) {
  const t = useT();
  const rules = useRules();
  const catalog = tariffCatalog(rules);
  const insertText = useActiveEditorInsert();
  const [search, setSearch] = useState("");
  const [tab, setTab] = useState<ReferenceTab>("data");
  const query = search.trim().toLowerCase();

  const entries = useMemo(() => {
    const all =
      tab === "data"
        ? dataEntries(catalog, context, t)
        : tab === "library"
          ? libraryEntries(catalog, context, t)
          : languageEntries(catalog, t);
    return query
      ? all.filter(
          (entry) =>
            entry.label.toLowerCase().includes(query) || entry.doc.toLowerCase().includes(query),
        )
      : all;
  }, [tab, catalog, context, t, query]);

  const rows = useMemo(
    () => rowsOf(entries, (entry) => insertText(entry.insert, entry.cursor)),
    [entries, insertText],
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="bg-card flex shrink-0 flex-col gap-2 border-b px-3 pt-3 pb-2.5">
        <Input
          id={REFERENCE_SEARCH_ID}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("Search the reference")}
          aria-label={t("Search the reference")}
          className="pr-8 pl-7"
          leftElement={<SearchLgIcon className="text-foreground-subtle size-3.5" />}
          rightElement={<Kbd className="mr-1">/</Kbd>}
        />
        <SegmentedControl<ReferenceTab>
          fullWidth
          aria-label={t("Reference")}
          value={tab}
          onValueChange={setTab}
          items={[
            { value: "data", label: t("Data") },
            { value: "library", label: t("Libraries") },
            { value: "language", label: t("Language") },
          ]}
        />
      </div>
      <VirtualRows
        rows={rows}
        estimateSize={ROW_HEIGHT}
        initialHeight={640}
        className="min-h-0 flex-1 px-2 pb-2"
        aria-label={t("Reference")}
        empty={
          <p className="text-foreground-muted px-4 py-6 text-center text-xs">
            {query ? t("Nothing matches “{0}”.", search) : t("Nothing to list here yet.")}
          </p>
        }
      />
    </div>
  );
}

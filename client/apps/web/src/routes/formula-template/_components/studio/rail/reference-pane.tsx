import {
  functionInsertion,
  type FunctionDoc,
  type VariableDoc,
} from "@/components/formula-editor/known-identifiers";
import {
  CATEGORY_LABELS,
  FUNCTION_CATEGORY_LABELS,
  categoryLabel,
} from "@/components/formula-editor/schema-labels";
import { useActiveEditorInsert } from "@/components/formula-editor/active-editor";
import { useFormulaSchema } from "@/hooks/use-formula-schema";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { PlusIcon, SearchLgIcon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useStudio } from "../studio-context";

export const REFERENCE_SEARCH_ID = "formula-reference-search";

function groupBy<T>(items: readonly T[], key: (item: T) => string): [string, T[]][] {
  const groups = new Map<string, T[]>();
  for (const item of items) {
    const groupKey = key(item);
    const existing = groups.get(groupKey);
    if (existing) existing.push(item);
    else groups.set(groupKey, [item]);
  }
  return [...groups.entries()];
}

function ReferenceRow({
  label,
  meta,
  description,
  example,
  onInsert,
}: {
  label: string;
  meta?: string;
  description: string;
  example?: string;
  onInsert: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onInsert}
      className="group ui-focus-ring hover:bg-surface-hover flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left"
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex min-w-0 items-baseline gap-2">
          <span className="truncate font-mono text-xs">{label}</span>
          {meta && <span className="text-foreground-subtle shrink-0 text-2xs">{meta}</span>}
        </span>
        {description && (
          <span className="text-foreground-muted line-clamp-1 text-xs">{description}</span>
        )}
        {example && (
          <code className="bg-sunken text-foreground-muted w-fit rounded-sm px-1.5 py-px font-mono text-2xs">
            {example}
          </code>
        )}
      </span>
      <PlusIcon className="text-foreground-subtle mt-0.5 size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100" />
    </button>
  );
}

/**
 * Every name the formula can use: the schema's variables, the template's
 * inputs and the functions, searchable with `/`. A row drops its name at the
 * cursor of the editor touched last.
 */
export function ReferencePane() {
  const t = useT();
  const { known } = useStudio();
  const { control } = useFormContext();
  const schemaId = (useWatch({ control, name: "schemaId" }) as string | undefined) || "shipment";
  const { isError: schemaUnavailable, refetch } = useFormulaSchema(schemaId);
  const insert = useActiveEditorInsert();
  const [search, setSearch] = useState("");
  const [tab, setTab] = useState<"variables" | "functions">("variables");
  const query = search.trim().toLowerCase();

  const variables = useMemo(
    () =>
      query
        ? known.variables.filter(
            (variable) =>
              variable.name.toLowerCase().includes(query) ||
              variable.description.toLowerCase().includes(query),
          )
        : known.variables,
    [known.variables, query],
  );
  const functions = useMemo(
    () =>
      query
        ? known.functions.filter(
            (fn) =>
              fn.name.toLowerCase().includes(query) || fn.description.toLowerCase().includes(query),
          )
        : known.functions,
    [known.functions, query],
  );

  const variableGroups = useMemo(
    () => groupBy<VariableDoc>(variables, (variable) => variable.category || "shipment"),
    [variables],
  );
  const functionGroups = useMemo(
    () => groupBy<FunctionDoc>(functions, (fn) => fn.category ?? ""),
    [functions],
  );
  const groups = tab === "variables" ? variableGroups : functionGroups;

  return (
    <div className="flex flex-col">
      <div className="bg-card sticky top-0 z-10 flex flex-col gap-2 border-b px-3 pt-3 pb-2.5">
        <Input
          id={REFERENCE_SEARCH_ID}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("Search variables and functions")}
          aria-label={t("Search variables and functions")}
          className="pr-8 pl-7"
          leftElement={<SearchLgIcon className="text-foreground-subtle size-3.5" />}
          rightElement={<Kbd className="mr-1">/</Kbd>}
        />
        <SegmentedControl<"variables" | "functions">
          fullWidth
          aria-label={t("Reference")}
          value={tab}
          onValueChange={setTab}
          items={[
            { value: "variables", label: t("Variables {0}", variables.length) },
            { value: "functions", label: t("Functions {0}", functions.length) },
          ]}
        />
        {schemaUnavailable && (
          <Alert variant="warning" size="sm" role="status">
            <AlertDescription>
              {t(
                "Showing the built-in reference; the live schema could not be loaded, so newer variables may be missing.",
              )}
            </AlertDescription>
            <AlertAction>
              <Button type="button" variant="ghost" size="xs" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertAction>
          </Alert>
        )}
      </div>
      <div className="flex flex-col gap-3 p-2">
        {tab === "variables"
          ? variableGroups.map(([category, items]) => (
              <div key={category} className="flex flex-col">
                <div className="text-foreground-subtle px-2 pb-1 text-xs font-medium">
                  {categoryLabel(category, CATEGORY_LABELS)}
                </div>
                {items.map((variable) => (
                  <ReferenceRow
                    key={variable.name}
                    label={variable.name}
                    meta={
                      variable.nullable
                        ? t("{0} · can be empty", t(variable.type))
                        : t(variable.type)
                    }
                    description={variable.description}
                    onInsert={() => insert(variable.name)}
                  />
                ))}
              </div>
            ))
          : functionGroups.map(([category, items]) => (
              <div key={category || "general"} className="flex flex-col">
                <div className="text-foreground-subtle px-2 pb-1 text-xs font-medium">
                  {categoryLabel(category, FUNCTION_CATEGORY_LABELS) || t("Functions")}
                </div>
                {items.map((fn) => (
                  <ReferenceRow
                    key={fn.name}
                    label={fn.signature}
                    description={fn.description}
                    example={fn.example}
                    onInsert={() => {
                      const insertion = functionInsertion(fn);
                      insert(insertion.text, insertion.cursor);
                    }}
                  />
                ))}
              </div>
            ))}
        {groups.length === 0 && (
          <p className="text-foreground-muted px-2 py-6 text-center text-xs">
            {t("Nothing matches “{0}”.", search)}
          </p>
        )}
      </div>
    </div>
  );
}

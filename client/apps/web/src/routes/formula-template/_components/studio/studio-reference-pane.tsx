import type {
  FunctionDoc,
  KnownIdentifiers,
  VariableDoc,
} from "@/components/formula-editor/known-identifiers";
import {
  CATEGORY_LABELS,
  FUNCTION_CATEGORY_LABELS,
  categoryLabel,
} from "@/components/formula-editor/schema-labels";
import { useFormulaSchema } from "@/hooks/use-formula-schema";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { BookOpenIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";

type StudioReferencePaneProps = {
  known: KnownIdentifiers;
  schemaId: string;
  onInsert: (text: string, cursorOffset?: number) => void;
};

function groupBy<T>(items: T[], key: (item: T) => string): [string, T[]][] {
  const groups = new Map<string, T[]>();
  for (const item of items) {
    const groupKey = key(item);
    const existing = groups.get(groupKey);
    if (existing) {
      existing.push(item);
    } else {
      groups.set(groupKey, [item]);
    }
  }
  return [...groups.entries()];
}

function VariableRow({
  variable,
  onInsert,
}: {
  variable: VariableDoc;
  onInsert: StudioReferencePaneProps["onInsert"];
}) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <button
            type="button"
            onClick={() => onInsert(variable.name)}
            className="group hover:bg-muted flex w-full items-center justify-between gap-2 rounded-md px-2 py-1 text-left"
          >
            <span className="truncate font-mono text-xs">{variable.name}</span>
            <span className="flex shrink-0 items-center gap-1.5">
              <Badge variant="outline" className="text-2xs px-1 py-0">
                {variable.type}
              </Badge>
              <PlusIcon className="text-muted-foreground size-3 opacity-0 transition-opacity group-hover:opacity-100" />
            </span>
          </button>
        }
      />
      <HoverCardContent side="left" className="w-72 space-y-1.5">
        <div className="font-mono text-sm font-semibold">{variable.name}</div>
        <p className="text-muted-foreground text-xs">
          {variable.description || "No description available."}
        </p>
        <div className="flex items-center gap-1.5">
          <Badge variant="outline" className="text-2xs">
            {variable.type}
          </Badge>
          {variable.nullable && (
            <Badge variant="outline" className="text-2xs">
              nullable
            </Badge>
          )}
        </div>
      </HoverCardContent>
    </HoverCard>
  );
}

function FunctionRow({
  fn,
  onInsert,
}: {
  fn: FunctionDoc;
  onInsert: StudioReferencePaneProps["onInsert"];
}) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <button
            type="button"
            onClick={() => onInsert(`${fn.name}()`, fn.name.length + 1)}
            className="group hover:bg-muted flex w-full items-center justify-between gap-2 rounded-md px-2 py-1 text-left"
          >
            <span className="truncate font-mono text-xs">{fn.signature}</span>
            <PlusIcon className="text-muted-foreground size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
          </button>
        }
      />
      <HoverCardContent side="left" className="w-72 space-y-1.5">
        <div className="font-mono text-sm font-semibold">{fn.signature}</div>
        <p className="text-muted-foreground text-xs">
          {fn.description || "No description available."}
        </p>
        {fn.example && (
          <code className="bg-muted block rounded px-2 py-1 font-mono text-xs">{fn.example}</code>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}

export function StudioReferencePane({ known, schemaId, onInsert }: StudioReferencePaneProps) {
  const [search, setSearch] = useState("");
  const [activeTab, setActiveTab] = useState<"variables" | "functions">("variables");
  const { isError: schemaUnavailable, refetch: refetchSchema } = useFormulaSchema(schemaId);

  const query = search.trim().toLowerCase();

  const filteredVariables = useMemo(
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

  const filteredFunctions = useMemo(
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
    () => groupBy(filteredVariables, (variable) => variable.category || "shipment"),
    [filteredVariables],
  );

  const functionGroups = useMemo(
    () => groupBy(filteredFunctions, (fn) => fn.category ?? ""),
    [filteredFunctions],
  );

  return (
    <div className="flex h-full flex-col">
      <div className="space-y-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <BookOpenIcon className="text-muted-foreground size-4" />
          <span className="text-sm font-semibold">Reference</span>
          <span className="text-muted-foreground text-2xs ml-auto">Click to insert</span>
        </div>
        {schemaUnavailable && (
          <div
            role="status"
            className="text-2xs flex items-center justify-between gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-2 py-1.5 text-amber-800 dark:text-amber-200"
          >
            <span>
              Showing the built-in reference; the live schema could not be loaded, so newer
              variables may be missing and flagged as unknown.
            </span>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="h-5 shrink-0"
              onClick={() => void refetchSchema()}
            >
              Retry
            </Button>
          </div>
        )}
        <div className="relative">
          <SearchIcon className="text-muted-foreground absolute top-1/2 left-2 size-3.5 -translate-y-1/2" />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search variables and functions..."
            className="h-7 pl-7 text-xs"
          />
        </div>
        <div className="flex gap-1">
          {(["variables", "functions"] as const).map((tab) => (
            <Button
              key={tab}
              type="button"
              variant={activeTab === tab ? "secondary" : "ghost"}
              size="xs"
              onClick={() => setActiveTab(tab)}
              className={cn("h-6 text-xs capitalize", activeTab !== tab && "text-muted-foreground")}
            >
              {tab} ({tab === "variables" ? filteredVariables.length : filteredFunctions.length})
            </Button>
          ))}
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="space-y-3 p-2">
          {activeTab === "variables" &&
            variableGroups.map(([category, variables]) => (
              <div key={category}>
                <div className="text-muted-foreground px-2 pb-1 text-xs font-medium tracking-wide uppercase">
                  {categoryLabel(category, CATEGORY_LABELS)}
                </div>
                <div className="space-y-0.5">
                  {variables.map((variable) => (
                    <VariableRow key={variable.name} variable={variable} onInsert={onInsert} />
                  ))}
                </div>
              </div>
            ))}

          {activeTab === "functions" &&
            functionGroups.map(([category, functions]) => (
              <div key={category || "general"}>
                <div className="text-muted-foreground px-2 pb-1 text-xs font-medium tracking-wide uppercase">
                  {categoryLabel(category, FUNCTION_CATEGORY_LABELS) || "Functions"}
                </div>
                <div className="space-y-0.5">
                  {functions.map((fn) => (
                    <FunctionRow key={fn.name} fn={fn} onInsert={onInsert} />
                  ))}
                </div>
              </div>
            ))}

          {((activeTab === "variables" && filteredVariables.length === 0) ||
            (activeTab === "functions" && filteredFunctions.length === 0)) && (
            <p className="text-muted-foreground px-2 py-6 text-center text-xs">
              Nothing matches &quot;{search}&quot;
            </p>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

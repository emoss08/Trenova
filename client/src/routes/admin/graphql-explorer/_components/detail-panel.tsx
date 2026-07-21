import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ShikiCodeBlock } from "@/components/ui/shiki-code-block";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type {
  CatalogFragment,
  CatalogOperation,
  CatalogSelection,
} from "@/types/graphql-catalog";
import { CheckIcon, CopyIcon, FileCodeIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { RunPanel } from "./run-panel";

type BadgeKind = "query" | "mutation" | "subscription" | "fragment";

function KindBadge({ kind }: { kind: BadgeKind }) {
  const styles: Record<BadgeKind, string> = {
    query: "border-sky-500/30 bg-sky-500/10 text-sky-600 dark:text-sky-400",
    mutation: "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400",
    subscription: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    fragment: "border-violet-500/30 bg-violet-500/10 text-violet-600 dark:text-violet-400",
  };
  return (
    <Badge variant="outline" className={cn("font-mono text-2xs uppercase", styles[kind])}>
      {kind}
    </Badge>
  );
}

function CopyButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(() => {
    void navigator.clipboard.writeText(value).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }, [value]);
  return (
    <Tooltip>
      <TooltipTrigger render={<Button size="icon-xs" variant="ghost" onClick={copy} />}>
        {copied ? <CheckIcon className="size-3.5 text-emerald-500" /> : <CopyIcon className="size-3.5" />}
      </TooltipTrigger>
      <TooltipContent>{copied ? "Copied" : label}</TooltipContent>
    </Tooltip>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
      {children}
    </span>
  );
}

function Chips({
  values,
  onSelect,
}: {
  values: string[];
  onSelect?: (name: string) => void;
}) {
  return (
    <div className="mt-1.5 flex flex-wrap gap-1.5">
      {values.map((value) =>
        onSelect ? (
          <button
            key={value}
            type="button"
            onClick={() => onSelect(value)}
            className="rounded-md border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-primary/5"
          >
            {value}
          </button>
        ) : (
          <span
            key={value}
            className="rounded-md border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-xs text-muted-foreground"
          >
            {value}
          </span>
        ),
      )}
    </div>
  );
}

function UsagesTab({ usages }: { usages: string[] }) {
  if (usages.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <FileCodeIcon className="size-6 text-muted-foreground" />
        <p className="mt-2 text-sm font-medium">No references found</p>
        <p className="mt-1 text-xs text-muted-foreground">
          This document is not referenced by any TypeScript source in <code>src/</code>.
        </p>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-1 py-1">
      <SectionLabel>
        {usages.length} file{usages.length === 1 ? "" : "s"}
      </SectionLabel>
      <ul className="mt-1 flex flex-col gap-1">
        {usages.map((usage) => (
          <li
            key={usage}
            className="flex items-center gap-2 rounded-md border border-border/60 bg-muted/30 px-2.5 py-1.5"
          >
            <FileCodeIcon className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate font-mono text-xs">{usage}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function MetaRow({ domain, sourceFile }: { domain: string; sourceFile: string }) {
  return (
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
      <Badge variant="secondary" className="font-normal">
        {domain}
      </Badge>
      <span className="font-mono">{sourceFile}</span>
    </div>
  );
}

function OperationDetail({
  operation,
  onSelect,
}: {
  operation: CatalogOperation;
  onSelect: (selection: CatalogSelection) => void;
}) {
  return (
    <Tabs defaultValue="definition" className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <KindBadge kind={operation.kind} />
          <h2 className="font-mono text-lg font-semibold">{operation.name}</h2>
          <CopyButton value={operation.name} label="Copy name" />
        </div>
        <MetaRow domain={operation.domain} sourceFile={operation.sourceFile} />
        {operation.hash && (
          <div className="flex items-center gap-1 text-2xs text-muted-foreground">
            <span className="font-mono">{operation.hash}</span>
            <CopyButton value={operation.hash} label="Copy hash" />
          </div>
        )}
      </div>

      <TabsList variant="underline" className="w-full justify-start border-b border-border">
        <TabsTrigger value="definition">Definition</TabsTrigger>
        <TabsTrigger value="run">Run</TabsTrigger>
        <TabsTrigger value="usages">Usages ({operation.usages.length})</TabsTrigger>
      </TabsList>

      <TabsContent value="definition" className="m-0 min-h-0 flex-1 overflow-auto">
        <div className="flex flex-col gap-4">
          {operation.variables.length > 0 && (
            <div>
              <SectionLabel>Variables</SectionLabel>
              <div className="mt-1.5 overflow-hidden rounded-md border">
                <table className="w-full text-xs">
                  <tbody>
                    {operation.variables.map((variable) => (
                      <tr key={variable.name} className="border-b last:border-b-0">
                        <td className="w-1/3 px-2.5 py-1.5 font-mono">${variable.name}</td>
                        <td className="px-2.5 py-1.5 font-mono text-muted-foreground">
                          {variable.type}
                          {variable.defaultValue !== null && (
                            <span className="text-muted-foreground/60">
                              {" = "}
                              {variable.defaultValue}
                            </span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {operation.rootFields.length > 0 && (
            <div>
              <SectionLabel>Root fields</SectionLabel>
              <Chips values={operation.rootFields} />
            </div>
          )}

          {operation.fragments.length > 0 && (
            <div>
              <SectionLabel>Fragments ({operation.fragments.length})</SectionLabel>
              <Chips
                values={operation.fragments}
                onSelect={(name) => onSelect({ kind: "fragment", name })}
              />
            </div>
          )}

          <div>
            <div className="flex items-center justify-between">
              <SectionLabel>Definition</SectionLabel>
              <CopyButton value={operation.sdl} label="Copy definition" />
            </div>
            <div className="mt-1.5">
              <ShikiCodeBlock code={operation.sdl} lang="graphql" darkTheme="vitesse-dark" />
            </div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="run" className="m-0 flex min-h-0 flex-1 flex-col">
        <RunPanel operation={operation} />
      </TabsContent>

      <TabsContent value="usages" className="m-0 min-h-0 flex-1 overflow-auto">
        <UsagesTab usages={operation.usages} />
      </TabsContent>
    </Tabs>
  );
}

function FragmentDetail({
  fragment,
  onSelect,
}: {
  fragment: CatalogFragment;
  onSelect: (selection: CatalogSelection) => void;
}) {
  return (
    <Tabs defaultValue="definition" className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <KindBadge kind="fragment" />
          <h2 className="font-mono text-lg font-semibold">{fragment.name}</h2>
          <span className="font-mono text-sm text-muted-foreground">on {fragment.typeCondition}</span>
          <CopyButton value={fragment.name} label="Copy name" />
        </div>
        <MetaRow domain={fragment.domain} sourceFile={fragment.sourceFile} />
      </div>

      <TabsList variant="underline" className="w-full justify-start border-b border-border">
        <TabsTrigger value="definition">Definition</TabsTrigger>
        <TabsTrigger value="usages">Usages ({fragment.usages.length})</TabsTrigger>
      </TabsList>

      <TabsContent value="definition" className="m-0 min-h-0 flex-1 overflow-auto">
        <div className="flex flex-col gap-4">
          {fragment.usedByOperations.length > 0 && (
            <div>
              <SectionLabel>Used by ({fragment.usedByOperations.length})</SectionLabel>
              <Chips
                values={fragment.usedByOperations}
                onSelect={(name) => onSelect({ kind: "operation", name })}
              />
            </div>
          )}

          {fragment.fragments.length > 0 && (
            <div>
              <SectionLabel>Nested fragments</SectionLabel>
              <Chips
                values={fragment.fragments}
                onSelect={(name) => onSelect({ kind: "fragment", name })}
              />
            </div>
          )}

          <div>
            <div className="flex items-center justify-between">
              <SectionLabel>Definition</SectionLabel>
              <CopyButton value={fragment.sdl} label="Copy definition" />
            </div>
            <div className="mt-1.5">
              <ShikiCodeBlock code={fragment.sdl} lang="graphql" darkTheme="vitesse-dark" />
            </div>
          </div>
        </div>
      </TabsContent>

      <TabsContent value="usages" className="m-0 min-h-0 flex-1 overflow-auto">
        <UsagesTab usages={fragment.usages} />
      </TabsContent>
    </Tabs>
  );
}

export function DetailPanel({
  operation,
  fragment,
  onSelect,
}: {
  operation: CatalogOperation | null;
  fragment: CatalogFragment | null;
  onSelect: (selection: CatalogSelection) => void;
}) {
  if (operation) {
    return <OperationDetail operation={operation} onSelect={onSelect} />;
  }
  if (fragment) {
    return <FragmentDetail fragment={fragment} onSelect={onSelect} />;
  }
  return (
    <div className="flex h-full flex-col items-center justify-center text-center">
      <FileCodeIcon className="size-8 text-muted-foreground" />
      <p className="mt-3 text-sm font-medium">Select an operation</p>
      <p className="mt-1 max-w-xs text-xs text-muted-foreground">
        Search by name, field, or domain to inspect a GraphQL query, mutation, or fragment.
      </p>
    </div>
  );
}

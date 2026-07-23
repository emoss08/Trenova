import { CopyIconButton } from "@/components/copy-icon-button";
import { Badge } from "@/components/ui/badge";
import { ShikiCodeBlock } from "@/components/ui/shiki-code-block";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import type { CatalogFragment, CatalogOperation, CatalogSelection } from "@/types/graphql-catalog";
import { Kbd } from "@/components/ui/kbd";
import { ScrollArea } from "@/components/ui/scroll-area";
import { FileCodeIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo } from "react";
import { catalog, referencedTypeNames } from "./catalog";
import { RunPanel } from "./run-panel";

type BadgeKind = "query" | "mutation" | "subscription" | "fragment";

const KIND_TEXT: Record<BadgeKind, string> = {
  query: "text-sky-600 dark:text-sky-400",
  mutation: "text-amber-600 dark:text-amber-400",
  subscription: "text-emerald-600 dark:text-emerald-400",
  fragment: "text-violet-600 dark:text-violet-400",
};

const KIND_TINT: Record<BadgeKind, string> = {
  query: "bg-sky-500/10",
  mutation: "bg-amber-500/10",
  subscription: "bg-emerald-500/10",
  fragment: "bg-violet-500/10",
};

function HashChip({ hash }: { hash: string }) {
  const short = `${hash.slice(0, 13)}…${hash.slice(-6)}`;
  return (
    <span
      title={hash}
      className="inline-flex items-center gap-0.5 rounded-md border bg-muted/40 py-0.5 pr-0.5 pl-1.5 font-mono text-2xs text-muted-foreground"
    >
      {short}
      <CopyIconButton value={hash} label="Copy hash" size="icon-xxs" />
    </span>
  );
}

function DocumentHeader({
  kind,
  name,
  suffix,
  domain,
  sourceFile,
  hash,
}: {
  kind: BadgeKind;
  name: string;
  suffix?: React.ReactNode;
  domain: string;
  sourceFile: string;
  hash?: string | null;
}) {
  return (
    <m.div
      key={`${kind}:${name}`}
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, ease: "easeOut" }}
      className="relative overflow-hidden rounded-lg border bg-card"
    >
      <div className="pointer-events-none absolute inset-0 [background-image:radial-gradient(currentColor_1px,transparent_1px)] [mask-image:linear-gradient(to_bottom,black,transparent)] [background-size:12px_12px] text-border" />
      <div
        className={cn(
          "pointer-events-none absolute -top-20 -right-10 size-56 rounded-full blur-3xl",
          KIND_TINT[kind],
        )}
      />
      <div className="relative flex flex-col gap-2 px-4 py-3">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className={cn("font-mono text-sm font-medium", KIND_TEXT[kind])}>{kind}</span>
          <h2 className="font-mono text-lg font-semibold tracking-tight">{name}</h2>
          {suffix}
          <CopyIconButton value={name} label="Copy name" />
        </div>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <Badge variant="secondary" className="font-normal">
            {domain}
          </Badge>
          <span className="font-mono text-xs text-muted-foreground">{sourceFile}</span>
          {hash && <HashChip hash={hash} />}
        </div>
      </div>
    </m.div>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
      {children}
    </span>
  );
}

function Chips({ values, onSelect }: { values: string[]; onSelect?: (name: string) => void }) {
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

function OperationDetail({
  operation,
  onSelect,
}: {
  operation: CatalogOperation;
  onSelect: (selection: CatalogSelection) => void;
}) {
  const inputTypeSdl = useMemo(() => {
    const names = referencedTypeNames(operation.variables);
    return names
      .map((name) => catalog.types[name]?.sdl)
      .filter(Boolean)
      .join("\n\n");
  }, [operation.variables]);

  return (
    <Tabs defaultValue="definition" className="flex min-h-0 flex-1 flex-col gap-3">
      <DocumentHeader
        kind={operation.kind}
        name={operation.name}
        domain={operation.domain}
        sourceFile={operation.sourceFile}
        hash={operation.hash}
      />

      <TabsList variant="underline" className="w-full justify-start border-b border-border">
        <TabsTrigger value="definition">Definition</TabsTrigger>
        <TabsTrigger value="run">Run</TabsTrigger>
        <TabsTrigger value="usages">Usages ({operation.usages.length})</TabsTrigger>
      </TabsList>

      <TabsContent value="definition" className="m-0 min-h-0 flex-1 overflow-hidden">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]>div]:block!">
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

            {inputTypeSdl && (
              <div>
                <SectionLabel>Input types</SectionLabel>
                <div className="mt-1.5">
                  <ShikiCodeBlock code={inputTypeSdl} lang="graphql" darkTheme="vitesse-dark" />
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
                <CopyIconButton value={operation.sdl} label="Copy definition" />
              </div>
              <div className="mt-1.5">
                <ShikiCodeBlock code={operation.sdl} lang="graphql" darkTheme="vitesse-dark" />
              </div>
            </div>
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent value="run" className="m-0 flex min-h-0 flex-1 flex-col">
        <RunPanel operation={operation} />
      </TabsContent>

      <TabsContent value="usages" className="m-0 min-h-0 flex-1 overflow-hidden">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]>div]:block!">
          <UsagesTab usages={operation.usages} />
        </ScrollArea>
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
      <DocumentHeader
        kind="fragment"
        name={fragment.name}
        suffix={
          <span className="font-mono text-sm text-muted-foreground">
            on {fragment.typeCondition}
          </span>
        }
        domain={fragment.domain}
        sourceFile={fragment.sourceFile}
      />

      <TabsList variant="underline" className="w-full justify-start border-b border-border">
        <TabsTrigger value="definition">Definition</TabsTrigger>
        <TabsTrigger value="usages">Usages ({fragment.usages.length})</TabsTrigger>
      </TabsList>

      <TabsContent value="definition" className="m-0 min-h-0 flex-1 overflow-hidden">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]>div]:block!">
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
                <CopyIconButton value={fragment.sdl} label="Copy definition" />
              </div>
              <div className="mt-1.5">
                <ShikiCodeBlock code={fragment.sdl} lang="graphql" darkTheme="vitesse-dark" />
              </div>
            </div>
          </div>
        </ScrollArea>
      </TabsContent>

      <TabsContent value="usages" className="m-0 min-h-0 flex-1 overflow-hidden">
        <ScrollArea className="h-full [&_[data-slot=scroll-area-viewport]>div]:block!">
          <UsagesTab usages={fragment.usages} />
        </ScrollArea>
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
    <div className="relative flex h-full flex-col items-center justify-center overflow-hidden text-center">
      <div className="pointer-events-none absolute inset-0 [background-image:radial-gradient(currentColor_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_at_center,black_20%,transparent_70%)] [background-size:14px_14px] text-border/70" />
      <div className="relative flex flex-col items-center">
        <FileCodeIcon className="size-8 text-muted-foreground" />
        <p className="mt-3 text-sm font-medium">Select an operation</p>
        <p className="mt-1 max-w-xs text-xs text-muted-foreground">
          Search by name, field, or domain to inspect a GraphQL query, mutation, or fragment.
        </p>
        <div className="mt-4 flex items-center gap-3 text-2xs text-muted-foreground/70">
          <span className="flex items-center gap-1">
            <Kbd>/</Kbd> search
          </span>
          <span className="flex items-center gap-1">
            <Kbd>↑↓</Kbd> navigate
          </span>
          <span className="flex items-center gap-1">
            <Kbd>⏎</Kbd> open
          </span>
        </div>
      </div>
    </div>
  );
}

import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  RuleVersion,
  RuleVersionStatus,
  SimulationRequest,
  SimulationResult,
} from "@/types/document-parsing-rule";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  ChevronDownIcon,
  EraserIcon,
  FlaskConicalIcon,
  InfoIcon,
  Loader2Icon,
  PlayIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { SimulationResultViewer } from "../simulation/simulation-result-viewer";

const VERSION_STATUS_BADGE: Record<RuleVersionStatus, BadgeVariant> = {
  Draft: "warning",
  Published: "active",
  Archived: "secondary",
};

function VersionSelect({
  versions,
  value,
  onChange,
}: {
  versions: RuleVersion[];
  value: string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);

  const selected = versions.find((v) => v.id === value);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        className="w-full"
        render={
          <Button
            variant="outline"
            className={cn(
              "group flex h-8 w-full items-center justify-between rounded-md border border-input bg-muted whitespace-nowrap hover:bg-muted/80",
              "px-1.5 py-2 text-xs ring-offset-background outline-hidden placeholder:text-muted-foreground",
              "data-pressed:border-brand data-pressed:ring-4 data-pressed:ring-brand/30",
              "transition-[border-color,box-shadow] duration-200 ease-in-out",
              "cursor-default",
            )}
          >
            <span
              className={cn(
                "flex min-w-0 flex-1 items-center gap-x-1.5 truncate font-normal",
                selected ? "text-foreground" : "text-muted-foreground",
              )}
            >
              {selected ? (
                <>
                  <Badge variant={VERSION_STATUS_BADGE[selected.status]} className="shrink-0">
                    {selected.status}
                  </Badge>
                  <span className="truncate">
                    Version {selected.versionNumber}
                    {selected.label ? ` — ${selected.label}` : ""}
                  </span>
                </>
              ) : (
                "Select a version..."
              )}
            </span>
            <ChevronDownIcon
              className={cn(
                "ml-auto size-3 opacity-50 transition-all duration-200 ease-in-out",
                open && "-rotate-180",
              )}
            />
          </Button>
        }
      />
      <PopoverContent
        className="border-input p-0"
        align="start"
        positionerClassName="min-w-(--anchor-width) rounded-lg dark"
      >
        <Command>
          <CommandInput placeholder="Search versions..." />
          <CommandList>
            <CommandEmpty>No versions found.</CommandEmpty>
            <CommandGroup>
              {versions.map((v) => (
                <CommandItem
                  key={v.id}
                  value={v.id}
                  onSelect={(val) => {
                    onChange(val === value ? "" : val);
                    setOpen(false);
                  }}
                  keywords={[
                    `Version ${v.versionNumber}`,
                    v.status,
                    v.label ?? "",
                  ]}
                >
                  {value === v.id && <CheckIcon className="size-3.5 shrink-0" />}
                  <Badge variant={VERSION_STATUS_BADGE[v.status]} className="shrink-0">
                    {v.status}
                  </Badge>
                  <span className="truncate text-xs">
                    Version {v.versionNumber}
                    {v.label ? ` — ${v.label}` : ""}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function HelpText({ children }: { children: React.ReactNode }) {
  return <p className="text-2xs text-foreground/70">{children}</p>;
}

function SectionHeading({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="space-y-0.5">
      <h3 className="text-sm font-medium">{title}</h3>
      <p className="text-2xs text-muted-foreground">{description}</p>
    </div>
  );
}

export default function SimulationTab({ ruleSetId }: { ruleSetId: string }) {
  const { data: versions, isLoading } = useQuery({
    ...queries.documentParsingRule.versions(ruleSetId),
  });

  const [selectedVersionId, setSelectedVersionId] = useState<string>("");
  const [fileName, setFileName] = useState("");
  const [providerFingerprint, setProviderFingerprint] = useState("");
  const [text, setText] = useState("");
  const [baselineJson, setBaselineJson] = useState("");
  const [result, setResult] = useState<SimulationResult | null>(null);

  const lineCount = useMemo(() => {
    if (!text) return 0;
    return text.split("\n").length;
  }, [text]);

  const simulateMutation = useMutation({
    mutationFn: async () => {
      const request: SimulationRequest = {
        fileName,
        text,
        pages: [],
        providerFingerprint,
        baseline: baselineJson ? JSON.parse(baselineJson) : undefined,
      };
      return apiService.documentParsingRuleService.simulate(
        selectedVersionId,
        request,
      );
    },
    onSuccess: (data) => {
      setResult(data);
    },
  });
  const { mutate: runSimulation, isPending, isError, error, reset } = simulateMutation;

  const handleSimulate = useCallback(() => {
    if (!selectedVersionId || !text.trim()) return;
    runSimulation();
  }, [selectedVersionId, text, runSimulation]);

  const handleClearResults = useCallback(() => {
    setResult(null);
    reset();
  }, [reset]);

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const eligibleVersions =
    versions?.filter(
      (v: RuleVersion) => v.status === "Draft" || v.status === "Published",
    ) ?? [];

  const canRun = selectedVersionId && text.trim();

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <FlaskConicalIcon className="size-4 text-muted-foreground" />
            <CardTitle>Rule Simulation</CardTitle>
          </div>
          <CardDescription>
            Run a rule version against sample document text to verify parsing behavior before
            publishing. Useful for testing new rules, debugging extraction issues, or comparing
            results against a known baseline.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* -- Document Input Section -- */}
          <div className="space-y-4">
            <SectionHeading
              title="Document Input"
              description="Select the rule version to test and provide the document text to parse."
            />

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-0.5">
                <Label className="required text-xs font-medium">Version</Label>
                <VersionSelect
                  versions={eligibleVersions}
                  value={selectedVersionId}
                  onChange={setSelectedVersionId}
                />
                <HelpText>Only Draft and Published versions are available for simulation.</HelpText>
              </div>
            </div>

            <div className="flex flex-col gap-0.5">
              <div className="flex items-center justify-between">
                <Label className="required text-xs font-medium">Document Text</Label>
                {text && (
                  <span className="text-2xs text-muted-foreground tabular-nums">
                    {lineCount} {lineCount === 1 ? "line" : "lines"}
                  </span>
                )}
              </div>
              <Textarea
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="Paste the full extracted document text here..."
                className="min-h-[200px] font-mono text-xs"
                minRows={8}
              />
              <HelpText>
                The raw text content extracted from the document. This is what the rule engine
                parses to extract fields, stops, and other structured data.
              </HelpText>
            </div>
          </div>

          {/* -- Optional Context Section -- */}
          <div className="space-y-4">
            <SectionHeading
              title="Optional Context"
              description="Additional metadata that helps the rule engine match and parse more accurately."
            />

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="flex flex-col gap-0.5">
                <Label className="text-xs font-medium">File Name</Label>
                <Input
                  value={fileName}
                  onChange={(e) => setFileName(e.target.value)}
                  placeholder="e.g. rate_confirmation_ch_robinson.pdf"
                />
                <HelpText>
                  Used by fileNameContains match rules. Provide the original document file name to
                  test file-name-based matching.
                </HelpText>
              </div>
              <div className="flex flex-col gap-0.5">
                <Label className="text-xs font-medium">Provider Fingerprint</Label>
                <Input
                  value={providerFingerprint}
                  onChange={(e) => setProviderFingerprint(e.target.value)}
                  placeholder="e.g. ch_robinson"
                />
                <HelpText>
                  A known provider identifier used by providerFingerprints match rules. Simulates
                  what the document intelligence pipeline would detect.
                </HelpText>
              </div>
            </div>

            <div className="flex flex-col gap-0.5">
              <Label className="text-xs font-medium">Baseline Analysis JSON</Label>
              <Textarea
                value={baselineJson}
                onChange={(e) => setBaselineJson(e.target.value)}
                placeholder='{"fields": {}, "stops": [], "overallConfidence": 0.95, ...}'
                className="min-h-[80px] font-mono text-xs"
                minRows={3}
              />
              <HelpText>
                Provide a previous analysis result as JSON to enable diff comparison. The simulation
                will show what changed between the baseline and the new candidate result — useful
                when iterating on rules to ensure changes produce the expected improvements.
              </HelpText>
            </div>
          </div>

          {/* -- Actions -- */}
          <div className="flex items-center gap-2">
            <Button
              type="button"
              onClick={handleSimulate}
              disabled={!canRun || isPending}
              className="gap-1.5"
            >
              {isPending ? (
                <>
                  <Loader2Icon className="size-4 animate-spin" />
                  Running...
                </>
              ) : (
                <>
                  <PlayIcon className="size-4" />
                  Run Simulation
                </>
              )}
            </Button>
            {result && (
              <Button
                type="button"
                variant="outline"
                onClick={handleClearResults}
                className="gap-1.5"
              >
                <EraserIcon className="size-3.5" />
                Clear Results
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {isError && (
        <Card className="border-destructive/50">
          <CardContent className="flex items-start gap-3 pt-4">
            <AlertTriangleIcon className="mt-0.5 size-4 shrink-0 text-destructive" />
            <div className="space-y-1">
              <p className="text-sm font-medium text-destructive">Simulation failed</p>
              <p className="text-xs text-muted-foreground">
                {error instanceof Error ? error.message : "An unexpected error occurred."}
              </p>
              <div className="flex items-start gap-1.5 pt-1">
                <InfoIcon className="mt-0.5 size-3 shrink-0 text-muted-foreground" />
                <p className="text-2xs text-muted-foreground">
                  Check that the selected version has valid rule configuration and the document text
                  is not empty. If the issue persists, verify the rule version status on the Versions
                  tab.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {result && <SimulationResultViewer result={result} />}
    </div>
  );
}

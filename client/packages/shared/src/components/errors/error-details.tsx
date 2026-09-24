import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDateInUserTimezone } from "@trenova/shared/lib/date";
import {
  type ErrorDescription,
  formatErrorReport,
  parseStackFrames,
} from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ChevronRightIcon, CopyIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

type CopyState = "idle" | "copied" | "failed";

const COPY_RESET_MS = 1600;

function useCopyText(): [CopyState, (text: string) => Promise<void>] {
  const [state, setState] = useState<CopyState>("idle");
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current) {
        clearTimeout(timer.current);
      }
    },
    [],
  );

  const copy = useCallback(async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setState("copied");
    } catch {
      setState("failed");
    }
    if (timer.current) {
      clearTimeout(timer.current);
    }
    timer.current = setTimeout(() => setState("idle"), COPY_RESET_MS);
  }, []);

  return [state, copy];
}

type ErrorReferenceProps = {
  traceId: string;
  className?: string;
};

// ErrorReference is the one identifier support can search the logs by. It is shown outside
// the details because asking someone to open a disclosure to read it to support loses it.
export function ErrorReference({ traceId, className }: ErrorReferenceProps) {
  const t = useT();
  const [state, copy] = useCopyText();

  return (
    <div className={cn("text-foreground-subtle flex items-center gap-1.5 text-xs", className)}>
      <span>{t("Reference")}</span>
      <code className="bg-sunken text-foreground-muted rounded-sm px-1.5 py-0.5 font-mono text-2xs">
        {traceId}
      </code>
      <Button
        variant="ghost"
        size="icon-xs"
        onClick={() => void copy(traceId)}
        aria-label={state === "copied" ? t("Reference copied") : t("Copy reference")}
      >
        {state === "copied" ? (
          <CheckIcon className="animate-confirm size-3" />
        ) : (
          <CopyIcon className="size-3" />
        )}
      </Button>
    </div>
  );
}

type ErrorDetailsProps = {
  description: ErrorDescription;
  title: string;
  occurredAt: Date;
  componentStack?: string | null;
  showDiagnostics?: boolean;
  defaultOpen?: boolean;
  className?: string;
};

// ErrorDetails holds what a person forwards to support and what an engineer reads. It is
// closed by default everywhere except a developer's own machine, where the stack is the
// point of looking.
export function ErrorDetails({
  description,
  title,
  occurredAt,
  componentStack,
  showDiagnostics = import.meta.env.DEV,
  defaultOpen = false,
  className,
}: ErrorDetailsProps) {
  const t = useT();
  const [state, copy] = useCopyText();
  const [open, setOpen] = useState(defaultOpen);

  const location = typeof window === "undefined" ? undefined : window.location.pathname;
  const frames = useMemo(
    () => (showDiagnostics ? parseStackFrames(description.stack) : []),
    [description.stack, showDiagnostics],
  );

  const report = useMemo(
    () =>
      formatErrorReport(description, {
        title,
        occurredAt,
        location: typeof window === "undefined" ? undefined : window.location.href,
        componentStack: showDiagnostics ? componentStack : null,
      }),
    [description, title, occurredAt, componentStack, showDiagnostics],
  );

  const copyLabel =
    state === "copied" ? t("Copied") : state === "failed" ? t("Couldn't copy") : t("Copy details");

  return (
    <Collapsible
      open={open}
      onOpenChange={setOpen}
      className={cn("w-full", className)}
      data-slot="error-details"
    >
      <div className="flex items-center justify-between gap-2">
        <CollapsibleTrigger
          render={
            <Button variant="ghost" size="xs" className="text-foreground-muted -ml-2">
              <ChevronRightIcon
                className={cn("size-3.5 transition-transform", open && "rotate-90")}
              />
              {t("Technical details")}
            </Button>
          }
        />
        <Button
          variant="ghost"
          size="xs"
          className="text-foreground-muted"
          onClick={() => void copy(report)}
          aria-live="polite"
        >
          {state === "copied" ? (
            <CheckIcon className="animate-confirm size-3.5" />
          ) : (
            <CopyIcon className="size-3.5" />
          )}
          {copyLabel}
        </Button>
      </div>

      <CollapsibleContent className="mt-2">
        <div className="bg-sunken border-border-subtle flex flex-col gap-3 rounded-lg border p-3 text-left">
          <DescriptionList layout="inline">
            <DescriptionItem label={t("Error")}>
              <span className="font-mono text-xs break-words">
                {description.name}: {description.message || t("No message")}
              </span>
            </DescriptionItem>
            {description.status ? (
              <DescriptionItem label={t("Status")} numeric>
                <span className="font-mono text-xs">{description.status}</span>
              </DescriptionItem>
            ) : null}
            {description.problemType ? (
              <DescriptionItem label={t("Problem")}>
                <span className="font-mono text-xs">{description.problemType}</span>
              </DescriptionItem>
            ) : null}
            {description.code ? (
              <DescriptionItem label={t("Code")}>
                <span className="font-mono text-xs">{description.code}</span>
              </DescriptionItem>
            ) : null}
            {description.traceId ? (
              <DescriptionItem label={t("Reference")}>
                <span className="font-mono text-xs break-all">{description.traceId}</span>
              </DescriptionItem>
            ) : null}
            {location ? (
              <DescriptionItem label={t("Page")}>
                <span className="font-mono text-xs break-all">{location}</span>
              </DescriptionItem>
            ) : null}
            <DescriptionItem label={t("Time")}>
              {formatDateInUserTimezone(occurredAt, { dateStyle: "medium", timeStyle: "medium" })}
            </DescriptionItem>
          </DescriptionList>

          {frames.length > 0 ? (
            <section className="border-border-subtle flex flex-col gap-1.5 border-t pt-3">
              <h4 className="text-foreground-subtle text-xs font-medium">{t("Stack trace")}</h4>
              <ol className="flex max-h-56 flex-col gap-0.5 overflow-auto font-mono text-2xs">
                {frames.map((frame, index) => (
                  <li
                    key={`${frame.file}:${frame.line}:${frame.column}:${index}`}
                    className="flex min-w-0 gap-2"
                  >
                    <span className="text-foreground shrink-0">{frame.fn}</span>
                    <span className="text-foreground-subtle truncate" title={frame.file}>
                      {frame.file}:{frame.line}:{frame.column}
                    </span>
                  </li>
                ))}
              </ol>
            </section>
          ) : null}

          {showDiagnostics && componentStack ? (
            <section className="border-border-subtle flex flex-col gap-1.5 border-t pt-3">
              <h4 className="text-foreground-subtle text-xs font-medium">{t("Component stack")}</h4>
              <pre className="text-foreground-muted max-h-56 overflow-auto font-mono text-2xs whitespace-pre-wrap">
                {componentStack.trim()}
              </pre>
            </section>
          ) : null}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

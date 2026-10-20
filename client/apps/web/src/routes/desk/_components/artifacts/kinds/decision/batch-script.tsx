import { ChevronRightIcon, CodeSnippet02Icon } from "@trenova/shared/components/icons";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import type { AssistantPlan } from "@/types/assistant";

const SCRIPT_LANGUAGE = "python";

/**
 * The script a batch of changes came from: what it was asked to work out and, on request,
 * the code that prepared every change below. Nothing in the batch has run; the script only
 * prepared it.
 */
export function BatchScript({ script }: { script: NonNullable<AssistantPlan["script"]> }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <div className="dk-ax-note flex flex-col gap-1">
      <p className="text-foreground font-medium">
        {script.title !== ""
          ? t("Written by a script: {0}", script.title)
          : t("Written by a script")}
      </p>
      {script.purpose !== "" && <p className="text-foreground-muted">{script.purpose}</p>}
      {script.code !== "" && (
        <Collapsible open={open} onOpenChange={setOpen}>
          <CollapsibleTrigger className="ui-focus-ring text-foreground-muted hover:text-foreground flex items-center gap-1.5 rounded-sm text-xs">
            <ChevronRightIcon
              className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")}
              aria-hidden
            />
            <CodeSnippet02Icon className="size-3.5 shrink-0" aria-hidden />
            {open ? t("Hide the script") : t("Show the script")}
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="scrollbar-overlay bg-sunken mt-1 max-h-72 overflow-auto rounded-sm">
              <HighlightedCode
                code={script.code}
                language={SCRIPT_LANGUAGE}
                className="w-max min-w-full px-3 py-2"
              />
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  );
}

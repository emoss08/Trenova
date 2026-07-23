import { Button } from "@/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { cn } from "@/lib/utils";
import type { SequenceConfig, SequenceConfigDocument } from "@/types/sequence-config";
import { CheckIcon, CopyIcon, InfoIcon } from "lucide-react";
import { memo, useMemo, useState } from "react";
import { useWatch } from "react-hook-form";
import { tokenLegend } from "./sequence-config-constants";
import { buildSequencePreview } from "./sequence-preview.utils";

type PreviewProps = {
  index: number;
  showTokens: boolean;
};

export const SequencePreview = memo(function SequencePreview({
  index,
  showTokens,
}: PreviewProps) {
  const config = useWatch<SequenceConfigDocument, `configs.${number}`>({
    name: `configs.${index}` as const,
  }) as SequenceConfig | undefined;

  const preview = useMemo(() => buildSequencePreview(config), [config]);
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!preview) return;
    try {
      await navigator.clipboard.writeText(preview);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // clipboard rejected — silent
    }
  };

  return (
    <div className="rounded-lg border border-border bg-muted/30 px-4 py-3.5">
      <div className="mb-1.5 flex items-center justify-between gap-3">
        <span className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
          Live Preview
        </span>
        <div className="flex items-center gap-1">
          {showTokens ? (
            <HoverCard>
              <HoverCardTrigger
                render={
                  <Button type="button" variant="ghost" size="xs" className="gap-1.5">
                    <InfoIcon className="size-3.5" />
                    Tokens
                  </Button>
                }
              />
              <HoverCardContent align="end" className="w-64">
                <div className="mb-1.5 text-xs font-medium">Custom format tokens</div>
                <ul className="grid grid-cols-2 gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  {tokenLegend.map(({ token, label }) => (
                    <li key={token} className="flex items-center gap-1.5">
                      <code className="rounded bg-muted px-1 py-0.5 font-mono text-[10px] text-foreground">
                        {token}
                      </code>
                      <span>{label}</span>
                    </li>
                  ))}
                </ul>
              </HoverCardContent>
            </HoverCard>
          ) : null}
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="gap-1.5"
            onClick={handleCopy}
            disabled={!preview}
          >
            {copied ? (
              <CheckIcon className="size-3.5 text-emerald-500" />
            ) : (
              <CopyIcon className="size-3.5" />
            )}
            {copied ? "Copied" : "Copy"}
          </Button>
        </div>
      </div>
      <code
        className={cn(
          "block font-mono text-xl font-semibold tracking-tight text-foreground",
        )}
      >
        {preview || "—"}
      </code>
      <p className="mt-1.5 text-xs text-muted-foreground">
        Representative sample — actual values increment sequentially.
      </p>
    </div>
  );
});

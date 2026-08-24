import { Button } from "@trenova/shared/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { cn } from "@trenova/shared/lib/utils";
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
    <div className="border-border bg-muted/30 rounded-lg border px-4 py-3.5">
      <div className="mb-1.5 flex items-center justify-between gap-3">
        <span className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
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
                <ul className="text-muted-foreground grid grid-cols-2 gap-x-3 gap-y-1 text-xs">
                  {tokenLegend.map(({ token, label }) => (
                    <li key={token} className="flex items-center gap-1.5">
                      <code className="bg-muted text-foreground rounded px-1 py-0.5 font-mono text-[10px]">
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
          "text-foreground block font-mono text-xl font-semibold tracking-tight",
        )}
      >
        {preview || "—"}
      </code>
      <p className="text-muted-foreground mt-1.5 text-xs">
        Representative sample — actual values increment sequentially.
      </p>
    </div>
  );
});

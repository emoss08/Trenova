"use client";

import { useT } from "@trenova/shared/i18n/use-t";
import * as React from "react";

import { Check, Copy, RefreshCw, Share2, ThumbsDown, ThumbsUp } from "lucide-react";

import { cn } from "@trenova/shared/lib/utils";

import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

type FeedbackType = "positive" | "negative" | null;

interface AiResponseActionsProps {
  content?: string;
  onRegenerate?: () => void;
  onFeedback?: (type: FeedbackType) => void;
  onShare?: () => void;
  compact?: boolean;
  className?: string;
}

export function AiResponseActions({
  content,
  onRegenerate,
  onFeedback,
  onShare,
  compact = false,
  className,
}: AiResponseActionsProps) {
  const t = useT();

  const [copied, setCopied] = React.useState(false);
  const [feedback, setFeedback] = React.useState<FeedbackType>(null);

  const handleCopy = React.useCallback(async () => {
    if (!content) return;
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [content]);

  const handleFeedback = React.useCallback(
    (type: FeedbackType) => {
      const newFeedback = feedback === type ? null : type;
      setFeedback(newFeedback);
      onFeedback?.(newFeedback);
    },
    [feedback, onFeedback],
  );

  const buttonSize = compact ? "size-7" : "size-8";
  const iconSize = compact ? "size-3.5" : "size-4";

  return (
    <div
      data-slot="ai-response-actions"
      role="toolbar"
      aria-label={t("Response actions")}
      className={cn(
        "bg-background inline-flex items-center gap-1 rounded-lg border p-1",
        className,
      )}
    >
      {content && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className={buttonSize}
                onClick={handleCopy}
                aria-label={copied ? "Copied" : "Copy response"}
              />
            }
          >
            {copied ? (
              <Check className={cn(iconSize, "text-green-500")} />
            ) : (
              <Copy className={iconSize} />
            )}
          </TooltipTrigger>
          <TooltipContent>
            <p>{copied ? t("Copied!") : t("Copy")}</p>
          </TooltipContent>
        </Tooltip>
      )}

      {onRegenerate && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className={buttonSize}
                onClick={onRegenerate}
                aria-label={t("Regenerate response")}
              />
            }
          >
            <RefreshCw className={iconSize} />
          </TooltipTrigger>
          <TooltipContent>
            <p>{t("Regenerate")}</p>
          </TooltipContent>
        </Tooltip>
      )}

      {onFeedback && (
        <>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    buttonSize,
                    feedback === "positive" && "bg-green-100 text-green-600 dark:bg-green-900/30",
                  )}
                  onClick={() => handleFeedback("positive")}
                  aria-label={t("Rate positive")}
                  aria-pressed={feedback === "positive"}
                />
              }
            >
              <ThumbsUp className={iconSize} />
            </TooltipTrigger>
            <TooltipContent>
              <p>{t("Good response")}</p>
            </TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    buttonSize,
                    feedback === "negative" && "bg-red-100 text-red-600 dark:bg-red-900/30",
                  )}
                  onClick={() => handleFeedback("negative")}
                  aria-label={t("Rate negative")}
                  aria-pressed={feedback === "negative"}
                />
              }
            >
              <ThumbsDown className={iconSize} />
            </TooltipTrigger>
            <TooltipContent>
              <p>{t("Bad response")}</p>
            </TooltipContent>
          </Tooltip>
        </>
      )}

      {onShare && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon"
                className={buttonSize}
                onClick={onShare}
                aria-label={t("Share response")}
              />
            }
          >
            <Share2 className={iconSize} />
          </TooltipTrigger>
          <TooltipContent>
            <p>{t("Share")}</p>
          </TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

export type { AiResponseActionsProps, FeedbackType };

"use client";

import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

interface Suggestion {
  label: string;
  prompt: string;
}

interface AiSuggestedActionsProps {
  suggestions: Suggestion[];
  onSelect?: (prompt: string) => void;
  className?: string;
}

function AiSuggestedActions({ suggestions, onSelect, className }: AiSuggestedActionsProps) {
  const t = useT();

  return (
    <div
      data-slot="ai-suggested-actions"
      className={cn("grid gap-2 font-mono sm:grid-cols-2", className)}
    >
      {suggestions.map((suggestion, index) => (
        <button
          type="button"
          key={suggestion.prompt}
          onClick={() => onSelect?.(suggestion.prompt)}
          className="bg-background hover:bg-muted border p-3 text-left text-xs transition-colors"
          style={{
            animationDelay: `${index * 50}ms`,
          }}
        >
          {t(suggestion.label)}
        </button>
      ))}
    </div>
  );
}

export { AiSuggestedActions };
export type { AiSuggestedActionsProps, Suggestion };

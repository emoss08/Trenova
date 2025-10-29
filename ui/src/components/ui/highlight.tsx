import { memo } from "react";

interface HighlightProps {
  text: string;
  highlight?: string;
  color?: string;
  className?: string;
}

function Highlight({ text, highlight = "", className }: HighlightProps) {
  if (!highlight.trim() || !text) {
    return <span className={className}>{text}</span>;
  }

  const escapeRegExp = (value: string) =>
    value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

  const safe = escapeRegExp(highlight);
  const parts = text.split(new RegExp(`(${safe})`, "gi"));

  return (
    <span className={className}>
      {parts.map((part, i) =>
        part.toLowerCase() === highlight.toLowerCase() ? (
          <span
            key={i}
            className="bg-yellow-400/80 shrink-0 font-medium dark:bg-yellow-400/40"
          >
            {part}
          </span>
        ) : (
          part
        ),
      )}
    </span>
  );
}

Highlight.displayName = "Highlight";

export default memo(Highlight);

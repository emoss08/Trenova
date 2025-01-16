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

  const parts = text.split(new RegExp(`(${highlight})`, "gi"));

  return (
    <span className={className}>
      {parts.map((part, i) =>
        part.toLowerCase() === highlight.toLowerCase() ? (
          <span
            key={i}
            className="bg-yellow-400/80 font-medium dark:bg-yellow-400/40"
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

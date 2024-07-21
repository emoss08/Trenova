import { memo, useMemo } from "react";

function Highlight({ text, highlight }: { text: string; highlight: string }) {
  const highlightedParts = useMemo(() => {
    if (!highlight.trim()) {
      return [{ text, highlighted: false }];
    }

    const regex = new RegExp(`(${highlight})`, "gi");
    const parts = text.split(regex);

    return parts.map((part) => ({
      text: part,
      highlighted: regex.test(part),
    }));
  }, [text, highlight]);

  return (
    <span>
      {highlightedParts.map((part, i) =>
        part.highlighted ? (
          <mark key={i} className="bg-blue-300">
            {part.text}
          </mark>
        ) : (
          <span key={i}>{part.text}</span>
        ),
      )}
    </span>
  );
}

Highlight.displayName = "Highlight";

export default memo(Highlight);

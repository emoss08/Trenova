import { Fragment } from "react";

/** Nova's words with the names it wrapped in backticks set as code. */
export function TickedText({ text }: { text: string }) {
  const parts = text.split(/(`[^`\n]+`)/g);
  return (
    <>
      {parts.map((part, index) =>
        part.length > 2 && part.startsWith("`") && part.endsWith("`") ? (
          <code
            // oxlint-disable-next-line react/no-array-index-key -- the split is positional and never reorders
            key={index}
            className="bg-sunken rounded-sm px-1 py-px font-mono"
          >
            {part.slice(1, -1)}
          </code>
        ) : (
          // oxlint-disable-next-line react/no-array-index-key -- the split is positional and never reorders
          <Fragment key={index}>{part}</Fragment>
        ),
      )}
    </>
  );
}

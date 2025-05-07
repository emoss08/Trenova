import { cn } from "@/lib/utils";
import type { DiffLineProps } from "@/types/json-viewer";
import { useMemo } from "react";
import { SensitiveBadge } from "../ui/sensitive-badge";

export function DiffLine({ line, lineNumber, type }: DiffLineProps) {
  const bgColor = useMemo(() => {
    if (type === "added") {
      return "bg-green-50 dark:bg-green-950/40";
    } else if (type === "removed") {
      return "bg-red-50 dark:bg-red-950/40";
    }
    return "";
  }, [type]);

  // * Define text colors based on type
  const textColor = useMemo(() => {
    if (type === "added") {
      return "text-green-600 dark:text-green-400";
    } else if (type === "removed") {
      return "text-red-600 dark:text-red-400";
    }
    return "text-foreground";
  }, [type]);

  // * Add a symbol at the beginning based on type
  const linePrefix = useMemo(() => {
    if (type === "added") {
      return <span className="text-green-600 dark:text-green-400 mr-2">+</span>;
    } else if (type === "removed") {
      return <span className="text-red-600 dark:text-red-400 mr-2">-</span>;
    }
    return <span className="mr-2"> </span>;
  }, [type]);

  const syntaxHighlightedLine = useMemo(() => {
    if (!line) return null;

    // Handle structural lines (only brackets or commas) with muted styling
    if (/^\s*[{}[\],]\s*$/.test(line)) {
      const indentMatch = line.match(/^\s*/);
      const indentation = indentMatch ? indentMatch[0] : "";
      const trimmedContent = line.trim();

      return (
        <>
          <span className="whitespace-pre">{indentation}</span>
          <span className="text-muted-foreground">{trimmedContent}</span>
        </>
      );
    }

    // Check if the line contains sensitive data (masked with asterisks)
    if (line.includes(':"****"') || line.includes(': "****"')) {
      const parts = line.split(/(".*?"\s*:\s*"(\*+)")/).filter(Boolean);

      return (
        <>
          {parts.map((part, index) => {
            if (part.includes('"****"')) {
              return (
                <span key={index} className="inline-flex items-center">
                  <span
                    dangerouslySetInnerHTML={{
                      __html: part.replace(
                        '"****"',
                        '<span class="text-vitess-string">"****"</span>',
                      ),
                    }}
                  />
                  <SensitiveBadge />
                </span>
              );
            }
            return <span key={index}>{part}</span>;
          })}
        </>
      );
    }

    const parts = [];
    let currentIndex = 0;

    // Preserve indentation
    const indentMatch = line.match(/^\s*/);
    const indentation = indentMatch ? indentMatch[0] : "";
    if (indentation) {
      parts.push(
        <span key="indentation" className="whitespace-pre">
          {indentation}
        </span>,
      );
      currentIndex = indentation.length;
    }

    // * Match property keys and their quotes
    const keyRegex = /"([^"]+)"(?=\s*:)/g;
    let match;

    while ((match = keyRegex.exec(line)) !== null) {
      // * Add any text before the match
      if (match.index > currentIndex) {
        parts.push(
          <span key={`pre-${match.index}`}>
            {line.substring(currentIndex, match.index)}
          </span>,
        );
      }

      // * Add the property key with highlighting
      parts.push(
        <span key={`key-${match.index}`} className="text-vitess-node">
          {match[0]}
        </span>,
      );

      currentIndex = match.index + match[0].length;
    }

    // * Add any remaining text
    if (currentIndex < line.length) {
      const remainingText = line.substring(currentIndex);

      // Handle colon specially with muted styling
      if (remainingText.trim().startsWith(":")) {
        const colonIndex = remainingText.indexOf(":");
        parts.push(
          <span key="pre-colon">{remainingText.substring(0, colonIndex)}</span>,
        );
        parts.push(
          <span key="colon" className="text-muted-foreground">
            :
          </span>,
        );

        const afterColonText = remainingText.substring(colonIndex + 1);

        // * Highlight string values
        const stringValueRegex = /\s*"([^"]*)"/g;
        const valueMatch = stringValueRegex.exec(afterColonText);

        if (valueMatch) {
          const preValueText = afterColonText.substring(0, valueMatch.index);
          const valueText = valueMatch[0];
          const postValueText = afterColonText.substring(
            valueMatch.index + valueText.length,
          );

          parts.push(<span key="pre-value">{preValueText}</span>);

          // Check if it's a sensitive value
          if (valueText.includes('"****"')) {
            parts.push(
              <span key="value" className="inline-flex items-center">
                <span className="text-vitess-string">{valueText}</span>
                <span className="ml-2 text-xs px-1.5 py-0.5 bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400 rounded-sm font-medium">
                  Sensitive
                </span>
              </span>,
            );
          } else if (
            valueText.match(
              /: "(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|0x[0-9a-fA-F]+)"/,
            )
          ) {
            // Handle numeric values in strings with comprehensive pattern matching:
            // - Regular numbers with optional decimal: 123, -123, 123.456, -123.456
            // - Scientific notation: 1.23e+4, -1.23e-4
            // - Hexadecimal: 0xFF, 0xA1B2C3
            const numMatch = valueText.match(
              /: "(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|0x[0-9a-fA-F]+)"/,
            );
            const numValue = numMatch ? numMatch[1] : "";

            parts.push(
              <span key="value">
                : <span className="text-vitess-string">&quot;</span>
                <span className="text-vitess-number">{numValue}</span>
                <span className="text-vitess-string">&quot;</span>
              </span>,
            );
          } else if (valueText.match(/: "(-?\d+\.?\d*)"/)) {
            // Handle numeric values in strings - detect by pattern not by field name
            const numMatch = valueText.match(/: "(-?\d+\.?\d*)"/);
            const numValue = numMatch ? numMatch[1] : "";

            parts.push(
              <span key="value">
                : <span className="text-vitess-string">&quot;</span>
                <span className="text-vitess-number">{numValue}</span>
                <span className="text-vitess-string">&quot;</span>
              </span>,
            );
          } else {
            parts.push(
              <span key="value" className="text-vitess-string">
                {valueText}
              </span>,
            );
          }

          // Handle comma in postValueText separately
          if (postValueText && postValueText.trim().startsWith(",")) {
            const commaIndex = postValueText.indexOf(",");
            parts.push(
              <span key="pre-comma">
                {postValueText.substring(0, commaIndex)}
              </span>,
            );
            parts.push(
              <span key="comma" className="text-muted-foreground">
                ,
              </span>,
            );
            parts.push(
              <span key="post-comma">
                {postValueText.substring(commaIndex + 1)}
              </span>,
            );
          } else {
            parts.push(<span key="post-value">{postValueText}</span>);
          }
        } else {
          // * Highlight other values (numbers, booleans, null)
          const formattedText = afterColonText
            .replace(
              /(\s*-?\d+(\.\d+)?)([,]?)/g,
              (_match, numPart, _decimalPart, comma) => {
                if (comma) {
                  return `<span class="text-vitess-number">${numPart}</span><span class="text-muted-foreground">${comma}</span>`;
                }
                return `<span class="text-vitess-number">${numPart}</span>`;
              },
            )
            .replace(
              /(\s*(?:true|false))([,]?)/g,
              (_match, boolPart, comma) => {
                if (comma) {
                  return `<span class="text-vitess-boolean">${boolPart}</span><span class="text-muted-foreground">${comma}</span>`;
                }
                return `<span class="text-vitess-boolean">${boolPart}</span>`;
              },
            )
            .replace(/(\s*null)([,]?)/g, (_match, nullPart, comma) => {
              if (comma) {
                return `<span class="text-gray-500">${nullPart}</span><span class="text-muted-foreground">${comma}</span>`;
              }
              return `<span class="text-gray-500">${nullPart}</span>`;
            });

          if (formattedText !== afterColonText) {
            parts.push(
              <span
                key="values"
                dangerouslySetInnerHTML={{ __html: formattedText }}
              />,
            );
          } else {
            // Handle any brackets in the remaining text
            const bracketRegex = /([{}[\]])/g;
            const bracketReplacements = afterColonText.replace(
              bracketRegex,
              '<span class="text-muted-foreground">$1</span>',
            );

            if (bracketReplacements !== afterColonText) {
              parts.push(
                <span
                  key="brackets"
                  dangerouslySetInnerHTML={{ __html: bracketReplacements }}
                />,
              );
            } else {
              parts.push(
                <span key="remaining-after-colon">{afterColonText}</span>,
              );
            }
          }
        }
      } else {
        // Handle any brackets in the remaining text
        const bracketRegex = /([{}[\]])/g;
        const bracketReplacements = remainingText.replace(
          bracketRegex,
          '<span class="text-muted-foreground">$1</span>',
        );

        if (bracketReplacements !== remainingText) {
          parts.push(
            <span
              key="brackets"
              dangerouslySetInnerHTML={{ __html: bracketReplacements }}
            />,
          );
        } else {
          parts.push(<span key="remaining">{remainingText}</span>);
        }
      }
    }

    return parts.length > 0 ? parts : line;
  }, [line]);

  return (
    <div className={cn("flex py-1 px-2", bgColor)}>
      <span className="w-8 text-muted-foreground text-xs font-mono pr-2 text-right select-none">
        {lineNumber}
      </span>
      <div
        className={cn(
          "font-mono text-sm flex-1 whitespace-pre-wrap",
          textColor,
        )}
      >
        {linePrefix}
        {syntaxHighlightedLine}
      </div>
    </div>
  );
}

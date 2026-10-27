import {
  HighlightedCode,
  type HighlightedCodeProps,
} from "@trenova/shared/components/ui/highlighted-code";
import { ladingTokens } from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import { useMemo } from "react";
import { ladingCodeSpans } from "./lading-highlight";
import { unlessRefused } from "./lading-service";

export type LadingCodeProps = Omit<HighlightedCodeProps, "language" | "spans">;

/**
 * Lading source, read-only, in the editor's syntax colours from the EDI module's tokens; text
 * the lexer refuses is plain. It suspends until the module has loaded.
 */
export function LadingCode({ code, ...props }: LadingCodeProps) {
  const edi = useEdi();
  const spans = useMemo(
    () => ladingCodeSpans(unlessRefused(() => ladingTokens(edi, code), [])),
    [edi, code],
  );
  return <HighlightedCode {...props} code={code} language="lading" spans={spans} />;
}

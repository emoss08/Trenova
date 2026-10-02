import { AiMarkdown } from "@/components/elements/ai-markdown";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { downloadTextFile, slugify } from "@trenova/shared/lib/utils";
import { CheckIcon, CopyIcon, DownloadIcon } from "lucide-react";
import { useMemo } from "react";
import { documentFrom } from "./artifact-payloads";
import { ArtifactToolbar } from "./artifact-section";

/**
 * A write-up the agent published: a brief, a summary, a handover. It is
 * read as the agent wrote it, and it leaves the conversation as text: copied
 * into an email or saved as a markdown file.
 */
export function DocumentArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const { body } = useMemo(() => documentFrom(artifact), [artifact]);
  const { copy, isCopied } = useCopyToClipboard();

  if (body === "") {
    return <ArtifactNotice kind={artifact.kind}>{t("This document is empty.")}</ArtifactNotice>;
  }

  const fileName = `${slugify(artifact.title) || "document"}.md`;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ArtifactToolbar className="justify-end">
        <Button
          variant="ghost"
          size="xs"
          className="text-foreground-muted hover:text-foreground"
          onClick={() => void copy(body, { timeout: 2000, withToast: true })}
        >
          <span
            key={isCopied ? "copied" : "copy"}
            className={isCopied ? "animate-confirm flex" : "flex"}
          >
            {isCopied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
          </span>
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
        <Button
          variant="ghost"
          size="xs"
          className="text-foreground-muted hover:text-foreground"
          onClick={() => downloadTextFile(fileName, body, "text/markdown")}
        >
          <DownloadIcon className="size-3.5" />
          {t("Download")}
        </Button>
      </ArtifactToolbar>
      <div className="scrollbar-overlay min-h-0 flex-1 overflow-y-auto px-5 py-4 text-sm">
        <AiMarkdown content={body} />
      </div>
    </div>
  );
}

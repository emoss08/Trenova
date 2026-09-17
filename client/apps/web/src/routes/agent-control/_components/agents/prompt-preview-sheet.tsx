import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { apiService } from "@/services/api";
import type { SaveAgentDefinitionRequest } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { EyeIcon } from "lucide-react";
import { useState } from "react";

type PromptPreviewSheetProps = {
  /** Called when the sheet opens, so the preview reflects the form as it is now. */
  getRequest: () => SaveAgentDefinitionRequest;
  disabled?: boolean;
};

/**
 * The exact system prompt the model would receive: Trenova's boundary,
 * then the organization's instructions, then the tools. Seeing it removes
 * the guesswork about what "instructions" turn into.
 */
export function PromptPreviewSheet({ getRequest, disabled = false }: PromptPreviewSheetProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [request, setRequest] = useState<SaveAgentDefinitionRequest | null>(null);

  const previewQuery = useQuery({
    queryKey: ["agent-prompt-preview", request],
    queryFn: () =>
      apiService.agentDefinitionService.previewPrompt(request as SaveAgentDefinitionRequest),
    enabled: open && request !== null,
    staleTime: 0,
  });

  return (
    <>
      <Button
        type="button"
        size="xs"
        variant="outline"
        disabled={disabled}
        onClick={() => {
          setRequest(getRequest());
          setOpen(true);
        }}
      >
        <EyeIcon className="size-3.5" />
        {t("Preview prompt")}
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-2xl">
          <SheetHeader className="border-border border-b px-5 py-4">
            <SheetTitle>{t("What the model will read")}</SheetTitle>
            <SheetDescription>
              {t(
                "Trenova's boundary comes first and cannot be overridden. Your instructions follow and are authoritative for everything else.",
              )}
            </SheetDescription>
          </SheetHeader>
          <ScrollArea className="min-h-0 flex-1">
            <div className="px-5 py-4">
              {previewQuery.isLoading || previewQuery.isFetching ? (
                <TextShimmer as="p" className="text-sm">
                  {t("Composing the prompt…")}
                </TextShimmer>
              ) : previewQuery.isError ? (
                <p className="text-destructive text-sm">
                  {t("The preview could not be built. Check the form for errors and try again.")}
                </p>
              ) : (
                <pre className="bg-muted/40 rounded-lg p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap">
                  {previewQuery.data?.prompt ?? ""}
                </pre>
              )}
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>
    </>
  );
}

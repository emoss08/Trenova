import { documentTemplateEditorParser } from "@/app/workers/_components/pto/use-document-template-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { faEye } from "@fortawesome/pro-regular-svg-icons";
import { PanelRightOpen, Sparkles } from "lucide-react";
import { useQueryStates } from "nuqs";
import React from "react";
import { DocumentTemplateLivePreview } from "./document-template-live-preview";

export function DocumentTemplatePreview() {
  return (
    <TemplatePreviewInner>
      <TemplatePreviewHeader />
      <DocumentTemplateLivePreview />
    </TemplatePreviewInner>
  );
}

function TemplatePreviewInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col bg-muted/30">{children}</div>
  );
}

function TemplatePreviewHeader() {
  const [, setSearchParams] = useQueryStates(documentTemplateEditorParser);

  return (
    <Outer>
      <div className="flex items-center gap-2">
        <div className="flex size-8 items-center justify-center rounded-lg bg-green-500/10">
          <Icon icon={faEye} className="size-4 text-green-600" />
        </div>
        <h3 className="truncate text-sm font-semibold text-foreground dark:text-green-50">
          Live Preview
        </h3>
      </div>
      <div className="flex items-center gap-2">
        <Badge variant="active" withDot={false}>
          <Sparkles className="size-2.5" />
          Auto-updating
        </Badge>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-6"
                onClick={() => setSearchParams({ showPreview: false })}
              >
                <PanelRightOpen className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="left">Hide preview</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </Outer>
  );
}

function Outer({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-[55px] shrink-0 items-center justify-between border-b border-border bg-gradient-to-r from-green-500/5 to-transparent p-3">
      {children}
    </div>
  );
}

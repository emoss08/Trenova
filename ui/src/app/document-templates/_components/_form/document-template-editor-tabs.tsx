import {
  documentTemplateEditorParser,
  editorTabs,
} from "@/app/workers/_components/pto/use-document-template-state";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { cn } from "@/lib/utils";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { useFormContext } from "react-hook-form";
import { TemplatePresets } from "../template-presets";

export function DocumentTemplateEditorTabContent() {
  const { setValue } = useFormContext<DocumentTemplateSchema>();

  const handlePresetSelect = useCallback(
    (preset: {
      htmlContent: string;
      cssContent: string;
      headerHtml?: string;
      footerHtml?: string;
    }) => {
      setValue("htmlContent", preset.htmlContent, { shouldDirty: true });
      setValue("cssContent", preset.cssContent, { shouldDirty: true });
      if (preset.headerHtml) {
        setValue("headerHtml", preset.headerHtml, { shouldDirty: true });
      }
      if (preset.footerHtml) {
        setValue("footerHtml", preset.footerHtml, { shouldDirty: true });
      }
    },
    [setValue],
  );

  return (
    <ContentOuter>
      <DocumentTemplateEditorTabs />
      <Separator orientation="vertical" className="mx-2 h-6" />
      <TemplatePresets onSelect={handlePresetSelect} />
    </ContentOuter>
  );
}

function ContentOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-1">{children}</div>;
}

export function DocumentTemplateEditorTabs() {
  const [searchParams, setSearchParams] = useQueryStates(
    documentTemplateEditorParser,
  );

  return (
    <DocumentTemplateEditorTabsOuter>
      {editorTabs.map((tab) => (
        <TooltipProvider key={tab.id}>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={() => setSearchParams({ editorTab: tab.id })}
                className={cn(
                  "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-all",
                  searchParams.editorTab === tab.id
                    ? "bg-background text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {tab.label}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{tab.description}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
    </DocumentTemplateEditorTabsOuter>
  );
}

function DocumentTemplateEditorTabsOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex rounded-lg border border-border bg-muted/50 p-0.5">
      {children}
    </div>
  );
}

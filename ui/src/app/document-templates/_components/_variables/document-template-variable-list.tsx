import {
  documentTemplateEditorParser,
  EditorTab,
} from "@/app/workers/_components/pto/use-document-template-state";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { PanelRightClose, Wand2 } from "lucide-react";
import { useQueryStates } from "nuqs";
import React, { useCallback } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { VariablePalette } from "./variable-palette";

export function DocumentTemplateVariableList() {
  const [searchParams] = useQueryStates(documentTemplateEditorParser);
  const { control, setValue } = useFormContext<DocumentTemplateSchema>();
  const htmlContent = useWatch({ control, name: "htmlContent" });
  const cssContent = useWatch({ control, name: "cssContent" });
  const headerHtml = useWatch({ control, name: "headerHtml" });
  const footerHtml = useWatch({ control, name: "footerHtml" });

  const handleInsertVariable = useCallback(
    (syntax: string) => {
      const fieldMap: Record<EditorTab, keyof DocumentTemplateSchema> = {
        html: "htmlContent",
        css: "cssContent",
        header: "headerHtml",
        footer: "footerHtml",
      };
      const fieldName = fieldMap[searchParams.editorTab];

      let currentValue: string | undefined;
      switch (searchParams.editorTab) {
        case "html":
          currentValue = htmlContent;
          break;
        case "css":
          currentValue = cssContent;
          break;
        case "header":
          currentValue = headerHtml;
          break;
        case "footer":
          currentValue = footerHtml;
          break;
      }

      setValue(fieldName, (currentValue || "") + syntax, { shouldDirty: true });
    },
    [
      searchParams.editorTab,
      htmlContent,
      cssContent,
      headerHtml,
      footerHtml,
      setValue,
    ],
  );

  return (
    <DocumentTemplateVariableListOuter>
      <DocumentTemplateVariableListHeader />
      <VariablePalette onInsert={handleInsertVariable} />
    </DocumentTemplateVariableListOuter>
  );
}

function DocumentTemplateVariableListOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-full min-w-[321px] flex-col border-l border-border bg-muted/10">
      {children}
    </div>
  );
}

function DocumentTemplateVariableListHeader() {
  const [, setSearchParams] = useQueryStates(documentTemplateEditorParser);

  return (
    <div className="flex shrink-0 items-center justify-between border-b border-border bg-gradient-to-r from-primary/5 to-transparent p-2">
      <div className="flex items-center gap-2">
        <Wand2 className="size-4 text-primary" />
        <span className="text-sm font-medium">Variables</span>
      </div>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => setSearchParams({ showVariables: false })}
            >
              <PanelRightClose className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="left">Hide variables</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}

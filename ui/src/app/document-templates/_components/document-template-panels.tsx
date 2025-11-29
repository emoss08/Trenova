import { documentTemplateEditorParser } from "@/app/workers/_components/pto/use-document-template-state";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { faEye, faEyeSlash } from "@fortawesome/pro-regular-svg-icons";
import { Maximize2, Minimize2, Wand2 } from "lucide-react";
import { useQueryStates } from "nuqs";

export function DocumentTemplateEditorControls() {
  const [searchParams, setSearchParams] = useQueryStates(
    documentTemplateEditorParser,
  );

  return (
    <Outer>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant={searchParams.showVariables ? "secondary" : "ghost"}
              onClick={() =>
                setSearchParams({
                  showVariables: !searchParams.showVariables,
                })
              }
              className="size-full"
            >
              <Wand2 className="size-3.5" />
              Variables
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {searchParams.showVariables
              ? "Hide variable palette"
              : "Show variable palette"}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant={searchParams.showPreview ? "green" : "outline"}
              onClick={() =>
                setSearchParams({ showPreview: !searchParams.showPreview })
              }
              className="size-full"
            >
              <Icon
                icon={searchParams.showPreview ? faEyeSlash : faEye}
                className="size-3.5"
              />
              Preview
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {searchParams.showPreview ? "Hide preview" : "Show live preview"}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant={searchParams.isFullscreen ? "default" : "outline"}
              className="h-8.5 w-full"
              onClick={() =>
                setSearchParams({
                  isFullscreen: !searchParams.isFullscreen,
                })
              }
            >
              {searchParams.isFullscreen ? (
                <Minimize2 className="size-4" />
              ) : (
                <Maximize2 className="size-4" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {searchParams.isFullscreen ? "Exit fullscreen" : "Fullscreen"}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Outer>
  );
}

function Outer({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-2">{children}</div>;
}

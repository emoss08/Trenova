import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { useTableStore } from "@/stores/table-store";
import { ExtraAction } from "@/types/data-table";
import { PlusIcon, UploadIcon } from "lucide-react";
import { isValidElement, useCallback } from "react";
import { DataTableImportModal } from "./data-table-import-modal";

type DataTableCreateContentProps = {
  name: string;
  extraActions?: ExtraAction[];
  exportModelName: string;
  handleCreateClick: () => void;
};

export function DataTableCreateContent({
  name,
  handleCreateClick,
  extraActions,
  exportModelName,
}: DataTableCreateContentProps) {
  const [showImportModal, setShowImportModal] =
    useTableStore.use("showImportModal");

  const handleImportClick = useCallback(() => {
    setShowImportModal(true);
  }, [setShowImportModal]);

  return (
    <>
      <DataCreateContentInner>
        <Button
          title={`Create a new ${name} from scratch`}
          variant="ghost"
          className="flex size-full flex-col items-start gap-0.5 text-left"
          onClick={handleCreateClick}
        >
          <div className="flex items-center gap-2">
            <PlusIcon className="size-4" />
            <span>Add New {name}</span>
          </div>
          <div>
            <p className="text-xs font-normal text-muted-foreground">
              Create a new {name} from scratch
            </p>
          </div>
        </Button>
        <Button
          title={`Import existing ${name}s from a file`}
          variant="ghost"
          className="flex size-full flex-col items-start gap-0.5 text-left"
          onClick={handleImportClick}
        >
          <div className="flex items-center gap-2">
            <UploadIcon className="size-4" />
            <span>Import {name}s</span>
          </div>
          <div>
            <p className="text-xs font-normal text-muted-foreground">
              Import existing {name}s from a file
            </p>
          </div>
        </Button>
        {extraActions?.map((option) => (
          <Button
            key={option.label}
            title={option.description}
            variant="ghost"
            className="flex size-full flex-col items-start gap-0.5 text-left"
            onClick={option.onClick}
          >
            <div className="flex items-center gap-2">
              {option.icon && <Icon icon={option.icon} className="size-4" />}
              <span>{option.label}</span>
              {isValidElement(option.endContent) && option.endContent}
            </div>
            <div>
              <p className="text-xs font-normal text-muted-foreground">
                {option.description}
              </p>
            </div>
          </Button>
        ))}
      </DataCreateContentInner>
      {showImportModal && (
        <DataTableImportModal
          open={showImportModal}
          onOpenChange={setShowImportModal}
          name={name}
          exportModelName={exportModelName}
        />
      )}
    </>
  );
}

function DataCreateContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex w-full flex-col gap-1">{children}</div>;
}

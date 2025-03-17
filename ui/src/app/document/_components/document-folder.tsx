import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Icon } from "@/components/ui/icons";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { formatFileSize, upperFirst } from "@/lib/utils";
import { DocumentCountByResource } from "@/services/document";
import { Resource } from "@/types/audit-entry";
import { faFolder } from "@fortawesome/pro-solid-svg-icons";

export function DocumentFolder({
  folder,
  onFolderClick,
}: {
  folder: DocumentCountByResource;
  onFolderClick: (folder: Resource) => void;
}) {
  return (
    <ContextMenu>
      <ContextMenuTrigger>
        <div
          key={folder.resourceType}
          className="flex flex-col items-center p-4 border rounded-lg hover:bg-muted cursor-pointer transition-colors"
          onClick={() => onFolderClick(folder.resourceType)}
        >
          <div className="flex items-center gap-2">
            <Icon icon={faFolder} className="text-[38px] text-blue-500 mr-3" />
          </div>

          <div className="flex items-start flex-col gap-1">
            <h3 className="font-medium">{`${upperFirst(folder.resourceType)}s`}</h3>
            <p className="text-sm text-muted-foreground">
              {folder.count} sub-folder
              {folder.count !== 1 ? "s" : ""}
            </p>
          </div>
        </div>
      </ContextMenuTrigger>
      <ContextMenuContent className="p-0 w-full">
        <span className="flex items-center text-sm font-medium p-1 border-b border-border w-full">
          Properties
        </span>
        <div className="flex flex-col gap-2 p-1">
          <DocumentFolderRow header="File Size">
            {formatFileSize(folder.totalSize)}
          </DocumentFolderRow>
          <DocumentFolderRow header="Last Modified">
            {generateDateTimeStringFromUnixTimestamp(folder.lastModified)}
          </DocumentFolderRow>
        </div>
      </ContextMenuContent>
    </ContextMenu>
  );
}

export function DocumentFolderSkeleton() {
  return (
    <div className="p-4 border rounded-lg animate-pulse flex items-center">
      <div className="w-10 h-10 bg-muted rounded-full mr-3" />
      <div className="flex-1">
        <div className="h-4 bg-muted rounded-full w-24 mb-2" />
        <div className="h-4 bg-muted rounded-full w-16" />
      </div>
    </div>
  );
}

export function DocumentFolderRow({
  header,
  children,
}: {
  header: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-1 w-full">
      <span className="text-sm font-medium">{header}:</span>
      <span className="text-xs text-muted-foreground">{children}</span>
    </div>
  );
}

import { Icon } from "@/components/ui/icons";
import { type ResourceFolder } from "@/types/document";
import { faFolderImage } from "@fortawesome/pro-solid-svg-icons";

export function DocumentFolder({
  folder,
  handleFolderClick,
}: {
  folder: ResourceFolder;
  handleFolderClick: (folder: ResourceFolder) => void;
}) {
  return (
    <div
      key={folder.resourceId}
      className="p-4 border rounded-lg hover:bg-muted cursor-pointer transition-colors flex items-center"
      onClick={() => handleFolderClick(folder)}
    >
      <Icon icon={faFolderImage} className="text-3xl text-blue-500 mr-3" />
      <div>
        <h3 className="font-medium">{folder.resourceName}</h3>
        <p className="text-sm text-muted-foreground">
          {folder.documentCount} document
          {folder.documentCount !== 1 ? "s" : ""}
        </p>
      </div>
    </div>
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

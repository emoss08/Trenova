/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Icon } from "@/components/ui/icons";
import { queries } from "@/lib/queries";
import { ResourceSubFolder } from "@/services/document";
import { Resource } from "@/types/audit-entry";
import { faFolder } from "@fortawesome/pro-regular-svg-icons";
import { useSuspenseQuery } from "@tanstack/react-query";
import { DocumentFolderSkeleton } from "./document-folder";
import { DocumentList } from "./document-list";

export function SubFolderView({
  selectedFolder,
  onBackClick,
  selectedSubFolder,
  onSubFolderClick,
}: {
  selectedFolder: Resource;
  onBackClick: () => void;
  onSubFolderClick: (subFolder: string) => void;
  selectedSubFolder: string;
}) {
  const { data: subFolders, isLoading } = useSuspenseQuery({
    ...queries.document.resourceSubFolders(selectedFolder),
  });

  if (isLoading) {
    return <FolderLoadingView />;
  }

  return selectedSubFolder ? (
    <DocumentList
      resourceType={selectedFolder}
      resourceId={selectedSubFolder}
    />
  ) : (
    <>
      <div className="flex items-center mb-4">
        <button
          className="text-blue-500 hover:underline flex items-center gap-1"
          onClick={onBackClick}
        >
          ← Back to main folders
        </button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
        {subFolders.map((subfolder) => (
          <DocumentSubFolder
            key={subfolder.folderName}
            folder={subfolder}
            onSubFolderClick={onSubFolderClick}
          />
        ))}
      </div>
    </>
  );
}

export function DocumentSubFolder({
  folder,
  onSubFolderClick,
}: {
  folder: ResourceSubFolder;
  onSubFolderClick: (subFolder: string) => void;
}) {
  return (
    <button
      onClick={() => onSubFolderClick(folder.resourceId)}
      className="p-4 border rounded-lg hover:bg-muted cursor-pointer transition-colors flex items-center"
    >
      <Icon icon={faFolder} className="text-3xl text-blue-500 mr-3" />
      <div>
        <h3 className="font-medium">{folder.folderName}</h3>
        <p className="text-sm text-muted-foreground">
          {folder.count} document{folder.count !== 1 ? "s" : ""}
        </p>
      </div>
    </button>
  );
}

function FolderLoadingView() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
      {Array.from({ length: 5 }).map((_, index) => (
        <DocumentFolderSkeleton key={index} />
      ))}
    </div>
  );
}

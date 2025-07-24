/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { queries } from "@/lib/queries";
import { Resource } from "@/types/audit-entry";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { useEffect } from "react";
import { searchParams } from "../search";
import { DocumentFolder, DocumentFolderSkeleton } from "./document-folder";
import { SubFolderView } from "./document-sub-folder";

export default function FolderList() {
  const [selectedFolder, setSelectedFolder] = useQueryState(
    "selectedFolder",
    searchParams.selectedFolder,
  );
  const [selectedSubFolder, setSelectedSubFolder] = useQueryState(
    "selectedSubFolder",
    searchParams.selectedSubFolder,
  );

  useEffect(() => {
    if (!selectedFolder) {
      setSelectedSubFolder(null);
    }
  }, [selectedFolder, setSelectedSubFolder]);

  return selectedFolder ? (
    <SubFolderView
      selectedFolder={selectedFolder as Resource}
      selectedSubFolder={selectedSubFolder}
      onBackClick={() => setSelectedFolder(null)}
      onSubFolderClick={(subFolder) => setSelectedSubFolder(subFolder)}
    />
  ) : (
    <RootFolderView
      onFolderClick={(folderType) => setSelectedFolder(folderType)}
    />
  );
}

function RootFolderView({
  onFolderClick,
}: {
  onFolderClick: (folder: Resource) => void;
}) {
  const { data: rootFolders, isLoading } = useSuspenseQuery({
    ...queries.document.countByResource(),
  });

  if (isLoading) {
    return <FolderLoadingView />;
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
      {rootFolders.map((folder) => (
        <DocumentFolder
          key={folder.resourceType}
          folder={folder}
          onFolderClick={() => onFolderClick(folder.resourceType)}
        />
      ))}
    </div>
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

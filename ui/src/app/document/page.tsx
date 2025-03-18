import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";

import DocumentUploadExample from "./_components/example";
import FolderList from "./_components/folder-list";

export function Document() {
  return (
    <>
      <MetaTags title="Document Studio" description="Document Studio" />
      <LazyComponent>
        <DocumentUploadExample />
        <FolderList />
      </LazyComponent>
    </>
  );
}

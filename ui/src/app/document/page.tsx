/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";

import FolderList from "./_components/folder-list";

export function Document() {
  return (
    <>
      <MetaTags title="Document Studio" description="Document Studio" />
      <LazyComponent>
        <FolderList />
      </LazyComponent>
    </>
  );
}

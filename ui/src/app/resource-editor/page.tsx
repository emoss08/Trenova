/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { MetaTags } from "@/components/meta-tags";
import { http } from "@/lib/http-client";
import { SchemaInformation } from "@/types/resource-editor";
import { useQuery } from "@tanstack/react-query";

export function ResourceEditor() {
  const { data: results, isLoading } = useQuery({
    queryKey: ["resource-editor"],
    queryFn: () => {
      return http.get<SchemaInformation>("/resource-editor/table-schema/");
    },
  });

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-lg">Loading schema information...</p>
      </div>
    );
  }

  if (!results?.data) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-lg text-red-500">
          Failed to load schema information or no data available.
        </p>
      </div>
    );
  }

  return (
    <>
      <MetaTags title="Resource Editor" description="Resource Editor" />
      <p>We&apos;re rewriting this page to use a new editor.</p>
    </>
  );
}

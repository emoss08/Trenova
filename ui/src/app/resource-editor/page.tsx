import { MetaTags } from "@/components/meta-tags";
import { http } from "@/lib/http-client";
import { SchemaInformation } from "@/types/resource-editor";
import { useQuery } from "@tanstack/react-query";
import { SchemaSidebar } from "./_components/schema-sidebar";
import { SQLEditor } from "./_components/sql-editor";

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
      <div className="flex h-screen">
        <SchemaSidebar results={results} />
        <SQLEditor results={results} />
      </div>
    </>
  );
}

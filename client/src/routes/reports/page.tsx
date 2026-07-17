import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermission } from "@/hooks/use-permission";
import { cn } from "@/lib/utils";
import { Operation, Resource } from "@/types/permission";
import { HistoryIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { CannedGallery } from "./_components/canned-gallery";
import { ReportDefinitionGrid } from "./_components/report-definition-grid";

type LibraryTab = "library" | "gallery";

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "relative flex h-8 items-center rounded-md px-3 text-sm transition-colors",
        active ? "font-medium text-foreground" : "text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
      {active && (
        <span className="absolute inset-x-2 bottom-[-7px] h-0.5 rounded-full bg-primary" />
      )}
    </button>
  );
}

export function ReportsPage() {
  const navigate = useNavigate();
  const { allowed: canCreate } = usePermission(Resource.Report, Operation.Create);
  const [tab, setTab] = useState<LibraryTab>("library");
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 300);

  return (
    <PageLayout
      className="p-0"
      pageHeaderProps={{
        title: "Reports",
        description: "Build, run, and share reports over your organization's data",
        actions: (
          <div className="flex items-center gap-2">
            <Button variant="outline" render={<Link to="/reports/runs" />}>
              <HistoryIcon className="size-4" />
              Run History
            </Button>
            {canCreate && (
              <Button onClick={() => void navigate("/reports/builder")}>
                <PlusIcon className="size-4" />
                New Report
              </Button>
            )}
          </div>
        ),
      }}
    >
      <div className="flex items-center gap-3 border-b border-border px-4 pt-1 pb-1.5">
        <div className="flex items-center">
          <TabButton active={tab === "library"} onClick={() => setTab("library")}>
            My Reports
          </TabButton>
          <TabButton active={tab === "gallery"} onClick={() => setTab("gallery")}>
            Gallery
          </TabButton>
        </div>
        <div className="flex-1" />
        <Input
          className="h-7 w-64 pl-8 text-xs"
          placeholder={tab === "library" ? "Search reports..." : "Search gallery..."}
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </div>
      {tab === "library" ? (
        <ReportDefinitionGrid search={debouncedSearch} />
      ) : (
        <CannedGallery search={search} />
      )}
    </PageLayout>
  );
}

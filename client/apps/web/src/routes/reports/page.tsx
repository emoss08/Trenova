import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermission } from "@/hooks/use-permission";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { HistoryIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { Link, useNavigate } from "react-router";
import { CannedGallery } from "./_components/canned-gallery";
import { DashboardGallery } from "./_components/dashboard-gallery";
import { ReportDefinitionGrid } from "./_components/report-definition-grid";
import { useCreateDashboardAction } from "./_components/use-create-dashboard-action";
import {
  GALLERY_SORT_CHOICES,
  LIBRARY_SORT_CHOICES,
  REPORT_CATEGORY_FILTER_CHOICES,
  REPORT_STATUS_FILTER_CHOICES,
  reportsPageSearchParamsParser,
  reportSortOrders,
  reportStatusFilters,
  type ReportSortOrder,
  type ReportStatusFilter,
  type ReportTab,
} from "./reports-page-state";

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
        active ? "text-foreground font-medium" : "text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
      {active && (
        <span className="bg-primary absolute inset-x-2 bottom-[-7px] h-0.5 rounded-full" />
      )}
    </button>
  );
}

function FilterSelect({
  items,
  value,
  onValueChange,
  ariaLabel,
}: {
  items: { value: string; label: string }[];
  value: string;
  onValueChange: (value: string) => void;
  ariaLabel: string;
}) {
  return (
    <Select
      items={items}
      value={value}
      onValueChange={(next) => {
        if (next !== null) {
          onValueChange(next);
        }
      }}
    >
      <SelectTrigger className="bg-background h-7 text-xs" aria-label={ariaLabel}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
}

function searchPlaceholder(tab: ReportTab): string {
  switch (tab) {
    case "library":
      return "Search reports...";
    case "dashboards":
      return "Search dashboards...";
    default:
      return "Search gallery...";
  }
}

export function ReportsPage() {
  const navigate = useNavigate();
  const { allowed: canCreate } = usePermission(Resource.Report, Operation.Create);
  const [params, setParams] = useQueryStates(reportsPageSearchParamsParser);
  const debouncedQuery = useDebounce(params.query, 300);
  const createDashboard = useCreateDashboardAction();

  const isLibrary = params.tab === "library";
  const isDashboards = params.tab === "dashboards";
  const sortChoices = isLibrary ? LIBRARY_SORT_CHOICES : GALLERY_SORT_CHOICES;

  const switchTab = (tab: ReportTab) => {
    void setParams({
      tab,
      sortBy: tab !== "library" && params.sortBy === "last_run" ? "name_asc" : params.sortBy,
      status: tab === "library" ? params.status : "all",
    });
  };

  return (
    <PageLayout
      className="gap-y-0 p-0"
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
      <div className="border-border flex flex-wrap items-center gap-1.5 border-b px-4 pt-1 pb-1.5">
        <div className="flex items-center">
          <TabButton active={isLibrary} onClick={() => switchTab("library")}>
            My Reports
          </TabButton>
          <TabButton active={params.tab === "gallery"} onClick={() => switchTab("gallery")}>
            Gallery
          </TabButton>
          <TabButton active={isDashboards} onClick={() => switchTab("dashboards")}>
            Dashboards
          </TabButton>
        </div>
        <div className="flex-1" />
        <div className="flex shrink-0 flex-row items-center gap-0 text-center text-sm">
          <div className="border-input bg-muted text-muted-foreground flex h-7 items-center gap-1 rounded-s-lg rounded-e-none border border-r-0 px-1.5 text-xs font-medium focus:z-10">
            Sort By
          </div>
          <Select
            items={sortChoices}
            value={params.sortBy}
            onValueChange={(value) => {
              if (value !== null && (reportSortOrders as readonly string[]).includes(value)) {
                void setParams({ sortBy: value as ReportSortOrder });
              }
            }}
          >
            <SelectTrigger
              className="bg-background h-7 rounded-s-none rounded-e-lg text-xs"
              aria-label="Sort reports"
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {sortChoices.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        {!isDashboards && (
          <div className="shrink-0">
            <FilterSelect
              items={REPORT_CATEGORY_FILTER_CHOICES}
              value={params.category}
              onValueChange={(value) => void setParams({ category: value })}
              ariaLabel="Filter by category"
            />
          </div>
        )}
        {isLibrary && (
          <div className="shrink-0">
            <FilterSelect
              items={REPORT_STATUS_FILTER_CHOICES}
              value={params.status}
              onValueChange={(value) => {
                if ((reportStatusFilters as readonly string[]).includes(value)) {
                  void setParams({ status: value as ReportStatusFilter });
                }
              }}
              ariaLabel="Filter by status"
            />
          </div>
        )}
        <Input
          className={cn("h-7 pl-8 text-xs", isDashboards ? "w-44" : "w-64")}
          placeholder={searchPlaceholder(params.tab)}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          value={params.query}
          onChange={(event) => void setParams({ query: event.target.value })}
        />
        {isDashboards && canCreate && (
          <Button
            variant="outline"
            size="sm"
            className="h-7 shrink-0 text-xs"
            onClick={createDashboard.create}
            disabled={createDashboard.isPending}
          >
            <PlusIcon className="size-3.5" />
            New Dashboard
          </Button>
        )}
      </div>
      {isLibrary && (
        <ReportDefinitionGrid
          search={debouncedQuery}
          sortBy={params.sortBy}
          category={params.category}
          status={params.status}
        />
      )}
      {params.tab === "gallery" && (
        <CannedGallery search={params.query} sortBy={params.sortBy} category={params.category} />
      )}
      {isDashboards && <DashboardGallery search={debouncedQuery} sortBy={params.sortBy} />}
    </PageLayout>
  );
}

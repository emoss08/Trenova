import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useDebounce } from "@/hooks/use-debounce";
import { usePermission } from "@/hooks/use-permission";
import { cn } from "@/lib/utils";
import { Operation, Resource } from "@/types/permission";
import { HistoryIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { Link, useNavigate } from "react-router";
import { CannedGallery } from "./_components/canned-gallery";
import { ReportDefinitionGrid } from "./_components/report-definition-grid";
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
      <SelectTrigger className="h-7 bg-background text-xs" aria-label={ariaLabel}>
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

export function ReportsPage() {
  const navigate = useNavigate();
  const { allowed: canCreate } = usePermission(Resource.Report, Operation.Create);
  const [params, setParams] = useQueryStates(reportsPageSearchParamsParser);
  const debouncedQuery = useDebounce(params.query, 300);

  const isLibrary = params.tab === "library";
  const sortChoices = isLibrary ? LIBRARY_SORT_CHOICES : GALLERY_SORT_CHOICES;

  const switchTab = (tab: ReportTab) => {
    void setParams({
      tab,
      sortBy: tab === "gallery" && params.sortBy === "last_run" ? "name_asc" : params.sortBy,
      status: tab === "gallery" ? "all" : params.status,
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
      <div className="flex flex-wrap items-center gap-1.5 border-b border-border px-4 pt-1 pb-1.5">
        <div className="flex items-center">
          <TabButton active={isLibrary} onClick={() => switchTab("library")}>
            My Reports
          </TabButton>
          <TabButton active={!isLibrary} onClick={() => switchTab("gallery")}>
            Gallery
          </TabButton>
        </div>
        <div className="flex-1" />
        <div className="flex shrink-0 flex-row items-center gap-0 text-center text-sm">
          <div className="flex h-7 items-center gap-1 rounded-s-lg rounded-e-none border border-r-0 border-input bg-muted px-1.5 text-xs font-medium text-muted-foreground focus:z-10">
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
              className="h-7 rounded-s-none rounded-e-lg bg-background text-xs"
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
        <div className="shrink-0">
          <FilterSelect
            items={REPORT_CATEGORY_FILTER_CHOICES}
            value={params.category}
            onValueChange={(value) => void setParams({ category: value })}
            ariaLabel="Filter by category"
          />
        </div>
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
          className="h-7 w-64 pl-8 text-xs"
          placeholder={isLibrary ? "Search reports..." : "Search gallery..."}
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          value={params.query}
          onChange={(event) => void setParams({ query: event.target.value })}
        />
      </div>
      {isLibrary ? (
        <ReportDefinitionGrid
          search={debouncedQuery}
          sortBy={params.sortBy}
          category={params.category}
          status={params.status}
        />
      ) : (
        <CannedGallery search={params.query} sortBy={params.sortBy} category={params.category} />
      )}
    </PageLayout>
  );
}

import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import type { FuelDashboardEntry } from "@/lib/graphql/fuel-surcharge";
import { useQuery } from "@tanstack/react-query";
import { History, ListTree, Pencil, Plus } from "lucide-react";
import { useState } from "react";
import { IndexPanel } from "./index-panel";
import { PriceHistoryDrawer } from "./price-history-drawer";

export default function IndexSection() {
  const { data: entries, isLoading } = useQuery(queries.fuelSurcharge.dashboard());
  const [panelOpen, setPanelOpen] = useState(false);
  const [editingEntry, setEditingEntry] = useState<FuelDashboardEntry | null>(null);
  const [historyEntry, setHistoryEntry] = useState<FuelDashboardEntry | null>(null);

  const openCreate = () => {
    setEditingEntry(null);
    setPanelOpen(true);
  };

  const openEdit = (entry: FuelDashboardEntry) => {
    setEditingEntry(entry);
    setPanelOpen(true);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          DOE indices ingest automatically each week via the EIA integration — custom indices take
          manually entered weekly prices (e.g. Canadian FCA or contract-specific pegs)
        </p>
        <Button type="button" size="sm" onClick={openCreate} className="gap-1.5">
          <Plus className="size-3.5" />
          New Custom Index
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : !entries || entries.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
          <div className="flex size-12 items-center justify-center rounded-full bg-muted">
            <ListTree className="size-5 text-muted-foreground" />
          </div>
          <p className="mt-3 text-sm font-medium">No fuel indices</p>
          <p className="mt-1 max-w-md text-xs text-muted-foreground">
            Enable the EIA Fuel Prices integration to auto-provision all 11 DOE diesel series, or
            create a custom index for manual weekly entry.
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead className="bg-muted/60">
              <tr className="text-left text-xs text-muted-foreground">
                <th className="px-4 py-2.5 font-medium">Code</th>
                <th className="px-4 py-2.5 font-medium">Name</th>
                <th className="px-4 py-2.5 font-medium">Region</th>
                <th className="px-4 py-2.5 font-medium">Fuel</th>
                <th className="px-4 py-2.5 font-medium">Source</th>
                <th className="px-4 py-2.5 font-medium">Latest Price</th>
                <th className="px-4 py-2.5 font-medium">Week</th>
                <th className="px-4 py-2.5 font-medium">Status</th>
                <th className="px-4 py-2.5" />
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.index.id} className="border-t transition-colors hover:bg-muted/30">
                  <td className="px-4 py-2.5 font-medium">{entry.index.code}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{entry.index.name}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{entry.index.region || "—"}</td>
                  <td className="px-4 py-2.5 text-muted-foreground">{entry.index.fuelType}</td>
                  <td className="px-4 py-2.5">
                    <Badge
                      variant={entry.index.source === "EIA" ? "secondary" : "outline"}
                      className="text-2xs"
                    >
                      {entry.index.source === "EIA" ? "DOE / EIA" : "Custom"}
                    </Badge>
                  </td>
                  <td className="px-4 py-2.5 tabular-nums">
                    {entry.latest ? `$${Number(entry.latest.price).toFixed(3)}` : "—"}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground tabular-nums">
                    {entry.latest?.priceDate ?? "—"}
                  </td>
                  <td className="px-4 py-2.5">
                    <Badge
                      variant="outline"
                      className={
                        entry.index.isActive
                          ? "border-emerald-500/40 text-2xs text-emerald-600 dark:text-emerald-400"
                          : "text-2xs text-muted-foreground"
                      }
                    >
                      {entry.index.isActive ? "Active" : "Inactive"}
                    </Badge>
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex justify-end gap-1">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setHistoryEntry(entry)}
                        className="size-7 gap-1 p-0 text-muted-foreground hover:text-foreground"
                        title="Price history"
                      >
                        <History className="size-3.5" />
                      </Button>
                      {entry.index.source === "Custom" && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => openEdit(entry)}
                          className="size-7 p-0 text-muted-foreground hover:text-foreground"
                          title="Edit index"
                        >
                          <Pencil className="size-3.5" />
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <IndexPanel open={panelOpen} onOpenChange={setPanelOpen} entry={editingEntry} />
      <PriceHistoryDrawer entry={historyEntry} onOpenChange={() => setHistoryEntry(null)} />
    </div>
  );
}

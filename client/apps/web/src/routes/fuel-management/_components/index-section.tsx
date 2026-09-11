import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { queries } from "@/lib/queries";
import type { FuelDashboardEntry } from "@/lib/graphql/fuel-surcharge";
import { useQuery } from "@tanstack/react-query";
import { History, Pencil, Plus } from "lucide-react";
import { useState } from "react";
import { FuelIndicesEmpty } from "./fuel-management-empty";
import { IndexPanel } from "./index-panel";
import { PriceHistoryDrawer } from "./price-history-drawer";

export default function IndexSection() {
  const t = useT();

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
        <p className="text-muted-foreground text-sm">
          {t("DOE indices ingest automatically each week via the EIA integration — custom indices take manually entered weekly prices (e.g. Canadian FCA or contract-specific pegs)")}
        </p>
        <Button type="button" size="sm" onClick={openCreate} className="gap-1.5">
          <Plus className="size-3.5" />
          {t("New Custom Index")}
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-64" />
      ) : !entries || entries.length === 0 ? (
        <FuelIndicesEmpty
          title={t("No indices yet")}
          description={
            "Enable the EIA Fuel Prices integration and all eleven DOE diesel series are provisioned " +
            "for you, or add a custom index and enter its weekly price by hand."
          }
          onCreate={openCreate}
        />
      ) : (
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead className="bg-muted/60">
              <tr className="text-muted-foreground text-left text-xs">
                <th className="px-4 py-2.5 font-medium">{t("Code")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Name")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Region")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Fuel")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Source")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Latest Price")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Week")}</th>
                <th className="px-4 py-2.5 font-medium">{t("Status")}</th>
                <th className="px-4 py-2.5" />
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.index.id} className="hover:bg-muted/30 border-t transition-colors">
                  <td className="px-4 py-2.5 font-medium">{entry.index.code}</td>
                  <td className="text-muted-foreground px-4 py-2.5">{entry.index.name}</td>
                  <td className="text-muted-foreground px-4 py-2.5">{entry.index.region || "—"}</td>
                  <td className="text-muted-foreground px-4 py-2.5">{entry.index.fuelType}</td>
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
                  <td className="text-muted-foreground px-4 py-2.5 tabular-nums">
                    {entry.latest?.priceDate ?? "—"}
                  </td>
                  <td className="px-4 py-2.5">
                    <Badge
                      variant="outline"
                      className={
                        entry.index.isActive
                          ? "text-2xs border-emerald-500/40 text-emerald-600 dark:text-emerald-400"
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
                        className="text-muted-foreground hover:text-foreground size-7 gap-1 p-0"
                        title={t("Price history")}
                      >
                        <History className="size-3.5" />
                      </Button>
                      {entry.index.source === "Custom" && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => openEdit(entry)}
                          className="text-muted-foreground hover:text-foreground size-7 p-0"
                          title={t("Edit index")}
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

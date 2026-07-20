import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  addFuelIndexPrice,
  deleteFuelIndexPrice,
  type FuelDashboardEntry,
} from "@/lib/graphql/fuel-surcharge";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

function mostRecentMonday(): string {
  const now = new Date();
  const day = now.getUTCDay();
  const diff = (day - 1 + 7) % 7;
  const monday = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - diff),
  );
  return monday.toISOString().slice(0, 10);
}

export function PriceHistoryDrawer({
  entry,
  onOpenChange,
}: {
  entry: FuelDashboardEntry | null;
  onOpenChange: () => void;
}) {
  const queryClient = useQueryClient();
  const indexId = entry?.index.id ?? "";
  const isCustom = entry?.index.source === "Custom";

  const [newDate, setNewDate] = useState(mostRecentMonday);
  const [newPrice, setNewPrice] = useState("");

  const { data: history, isLoading } = useQuery({
    ...queries.fuelSurcharge.priceHistory(indexId, 52),
    enabled: !!entry,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: queries.fuelSurcharge.priceHistory(indexId, 52).queryKey,
    });
    void queryClient.invalidateQueries({
      queryKey: queries.fuelSurcharge.dashboard().queryKey,
    });
  };

  const { mutate: addPrice, isPending: isAdding } = useMutation({
    mutationFn: () =>
      addFuelIndexPrice({ fuelIndexId: indexId, priceDate: newDate, price: newPrice }),
    onSuccess: () => {
      toast.success("Weekly price added");
      setNewPrice("");
      invalidate();
    },
    onError: () => {
      toast.error("Could not add the price", {
        description: "Check the date (one price per week) and value",
      });
    },
  });

  const { mutate: removePrice } = useMutation({
    mutationFn: (id: string) => deleteFuelIndexPrice(id),
    onSuccess: () => {
      toast.success("Price removed");
      invalidate();
    },
    onError: () => {
      toast.error("Automatically ingested prices cannot be deleted");
    },
  });

  return (
    <Sheet open={!!entry} onOpenChange={(open) => !open && onOpenChange()}>
      <SheetContent className="flex w-full flex-col sm:max-w-md">
        <SheetHeader>
          <SheetTitle>{entry?.index.name}</SheetTitle>
          <SheetDescription>
            {isCustom
              ? "Manually entered weekly prices — enter the Monday date each price is effective for"
              : "Weekly DOE prices ingested automatically from the EIA API"}
          </SheetDescription>
        </SheetHeader>

        {isCustom && (
          <div className="flex items-end gap-2 rounded-lg border bg-muted/30 p-3">
            <div className="flex-1 space-y-1">
              <Label className="text-xs">Week (Monday)</Label>
              <Input
                type="date"
                value={newDate}
                onChange={(event) => setNewDate(event.target.value)}
              />
            </div>
            <div className="flex-1 space-y-1">
              <Label className="text-xs">Price ($/gal)</Label>
              <Input
                value={newPrice}
                onChange={(event) => setNewPrice(event.target.value)}
                placeholder="3.759"
                inputMode="decimal"
              />
            </div>
            <Button
              type="button"
              size="sm"
              onClick={() => addPrice()}
              disabled={isAdding || !newPrice || !newDate}
              className="gap-1"
            >
              <Plus className="size-3.5" />
              Add
            </Button>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border">
          {isLoading ? (
            <div className="space-y-2 p-3">
              {Array.from({ length: 8 }).map((_, index) => (
                <Skeleton key={index} className="h-8" />
              ))}
            </div>
          ) : !history || history.length === 0 ? (
            <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
              No prices recorded yet
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-muted/80 backdrop-blur">
                <tr className="text-left text-xs text-muted-foreground">
                  <th className="px-3 py-2 font-medium">Week</th>
                  <th className="px-3 py-2 font-medium">Price</th>
                  <th className="px-3 py-2 font-medium">Source</th>
                  <th className="px-3 py-2" />
                </tr>
              </thead>
              <tbody>
                {history.map((price) => (
                  <tr key={price.id} className="group border-t tabular-nums">
                    <td className="px-3 py-1.5">{price.priceDate}</td>
                    <td className="px-3 py-1.5">${Number(price.price).toFixed(3)}</td>
                    <td className="px-3 py-1.5">
                      <Badge
                        variant={price.isManual ? "outline" : "secondary"}
                        className="text-2xs"
                      >
                        {price.isManual ? "Manual" : "EIA"}
                      </Badge>
                    </td>
                    <td className="px-3 py-1.5 text-right">
                      {price.isManual && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => removePrice(price.id)}
                          className="size-6 p-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-destructive"
                        >
                          <Trash2 className="size-3" />
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

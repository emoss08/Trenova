import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  assignableOptions,
  decidingValue,
  hasSellPrice,
  marginTone,
  offerWindowLabel,
  optionIsPriced,
  shopOptionNote,
  shopStrategyExplanation,
  shopStrategyLabel,
  type MarginTone,
} from "@trenova/shared/lib/shop";
import type { ShopOption, ShopResult, ShopStrategy } from "@trenova/shared/types/rate";
import { CircleAlertIcon, LoaderCircleIcon, SearchIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

const STRATEGIES: ShopStrategy[] = ["LeastCost", "BestMargin", "GuideRank", "FastestAccept"];

const TONE_CLASS: Record<MarginTone, string> = {
  breach: "text-destructive",
  thin: "text-warning",
  healthy: "text-foreground",
  unknown: "text-muted-foreground",
};

type CarrierShoppingPanelProps = {
  readonly shipmentId: string;
  /** Fills the assignment form from the carrier somebody picked. */
  readonly onChoose: (option: ShopOption) => void;
};

/**
 * What each carrier on this lane would charge, ranked.
 *
 * The order is the server's and is deliberately not re-sorted here: a screen
 * sorting its own way would name a different winner from the stored result,
 * and the stored result is what somebody cites when asked why a carrier was
 * offered the load.
 *
 * Nothing is persisted while somebody is browsing. Every carrier glanced at
 * would otherwise leave a quote behind competing with the shipment's real one.
 */
export function CarrierShoppingPanel({ shipmentId, onChoose }: CarrierShoppingPanelProps) {
  const [strategy, setStrategy] = useState<ShopStrategy>("LeastCost");
  const [result, setResult] = useState<ShopResult | undefined>();

  const { mutate: runShop, isPending } = useMutation({
    mutationFn: (chosen: ShopStrategy) =>
      apiService.rateQuoteService.shop(shipmentId, { strategy: chosen }),
    onSuccess: setResult,
    onError: () => toast.error("Could not shop this lane"),
  });

  const run = useCallback(
    (chosen: ShopStrategy) => {
      setStrategy(chosen);
      runShop(chosen);
    },
    [runShop],
  );

  const sellPriced = hasSellPrice(result);
  const assignable = assignableOptions(result);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-end gap-2">
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">Rank by</span>
          <Select value={strategy} onValueChange={(value) => setStrategy(value as ShopStrategy)}>
            <SelectTrigger className="w-52">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STRATEGIES.map((each) => (
                <SelectItem key={each} value={each}>
                  {shopStrategyLabel(each)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={isPending}
          onClick={() => run(strategy)}
        >
          {isPending ? (
            <LoaderCircleIcon className="mr-1 size-3.5 animate-spin" />
          ) : (
            <SearchIcon className="mr-1 size-3.5" />
          )}
          Shop carriers
        </Button>
      </div>

      <p className="text-muted-foreground text-xs">{shopStrategyExplanation(strategy)}</p>

      {result?.warnings.map((warning) => (
        <Alert key={warning}>
          <CircleAlertIcon className="size-4" />
          <AlertDescription>{warning}</AlertDescription>
        </Alert>
      ))}

      {result && assignable.length === 0 && result.options.length > 0 && (
        <Alert variant="destructive">
          <CircleAlertIcon className="size-4" />
          <AlertDescription>
            No carrier on this lane has a contract that prices it. Write one, or enter the rate by
            hand.
          </AlertDescription>
        </Alert>
      )}

      {result && result.options.length > 0 && (
        <div className="overflow-x-auto rounded-md border">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="bg-muted/50 text-muted-foreground text-xs">
                <th className="border-b px-3 py-2 text-left font-medium">Carrier</th>
                <th className="border-b px-3 py-2 text-right font-medium">Cost</th>
                <th className="border-b px-3 py-2 text-right font-medium">Margin</th>
                <th className="border-b px-3 py-2 text-right font-medium">
                  {shopStrategyLabel(result.strategy)}
                </th>
                <th className="border-b px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {result.options.map((option) => {
                const priced = optionIsPriced(option);
                const note = shopOptionNote(option, sellPriced);
                const tone = marginTone(option.margin, sellPriced);

                return (
                  <tr key={option.carrierId}>
                    <td className="border-b px-3 py-2">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">
                          {option.carrierName || option.carrierId}
                        </span>
                        {option.guideRank > 0 && (
                          <Badge variant="secondary" className="text-2xs h-4.5">
                            Guide #{option.guideRank}
                          </Badge>
                        )}
                        {option.offerTtlSeconds > 0 && (
                          <span className="text-muted-foreground text-2xs">
                            {offerWindowLabel(option.offerTtlSeconds)} to accept
                          </span>
                        )}
                      </div>
                      {note && <p className="text-muted-foreground mt-0.5 text-xs">{note}</p>}
                    </td>
                    <td className="border-b px-3 py-2 text-right font-mono text-xs">
                      {priced ? `${option.cost} ${option.currency}` : "—"}
                    </td>
                    <td
                      className={`border-b px-3 py-2 text-right font-mono text-xs ${TONE_CLASS[tone]}`}
                    >
                      {priced && sellPriced ? `${option.margin.percent}%` : "—"}
                    </td>
                    <td className="border-b px-3 py-2 text-right font-mono text-xs">
                      {priced ? decidingValue(option, result.strategy) : "—"}
                    </td>
                    <td className="border-b px-3 py-2 text-right">
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        disabled={!priced}
                        onClick={() => onChoose(option)}
                      >
                        Use
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

import { reorderAIProviders } from "@/lib/graphql/ai-provider";
import { aiAnalytics } from "@/lib/queries/ai-analytics";
import { aiProvider } from "@/lib/queries/ai-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memo, type CSSProperties } from "react";
import { toast } from "sonner";
import { Ic } from "../kit/ic";
import { Mark } from "../kit/marks";
import { BOLD, display, moveAhead, type SameModelPair, type SeriesColors } from "./analytics-model";
import { AnalyticsCard } from "./card";
import type { AIAnalyticsProvider } from "@/lib/graphql/ai-analytics";

type SameModelCardProps = {
  pair: SameModelPair;
  colors: SeriesColors;
  /** Every provider, in routing order. */
  routing: readonly { id: string }[];
  canRoute: boolean;
};

/**
 * One model on two providers, figure by figure. The model is the same, so every gap is the
 * connection's; the better provider can be put ahead of the other in routing from here.
 */
export const SameModelCard = memo(function SameModelCard({
  pair,
  colors,
  routing,
  canRoute,
}: SameModelCardProps) {
  const t = useT();
  const rt = useRichT();
  const queryClient = useQueryClient();
  const { left, right, rows, wins } = pair;
  const leftId = left.providerId ?? "";
  const rightId = right.providerId ?? "";
  const ids = routing.map((provider) => provider.id);
  const routable = ids.includes(leftId) && ids.includes(rightId);
  const order = routable ? moveAhead(ids, leftId, rightId) : null;
  const reorder = useMutation({
    mutationFn: (ids: string[]) => reorderAIProviders(ids),
    onSuccess: async () => {
      toast.success(t("{0} now goes ahead of {1}", left.label, right.label));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: aiProvider._def }),
        queryClient.invalidateQueries({ queryKey: aiAnalytics._def }),
      ]);
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t("The routing could not be changed")),
  });
  const model = left.models[0] ?? "";

  return (
    <AnalyticsCard
      title={t("Same model, different provider")}
      description={t(
        "{0} runs on both. Since the model is the same, every gap here comes from the connection.",
        model,
      )}
      className="an-same"
    >
      <div className="an-bfh">
        <span>
          <ProviderMark provider={left.providers[0]} />
          <b>{left.label}</b>
        </span>
        <em className="mono">{t("vs")}</em>
        <span>
          <b>{right.label}</b>
          <ProviderMark provider={right.providers[0]} />
        </span>
      </div>
      <div className="an-bf">
        {rows.map((row, index) => (
          <div key={row.metric.key} className="an-bfr">
            <span className={cn("an-bfv mono", row.leftWins && "w")}>
              {display(t, row.metric, row.left)}
            </span>
            <span className="an-bfb l">
              <i
                className={cn(row.leftWins && "w")}
                style={
                  {
                    width: `${(row.left / row.max) * 100}%`,
                    "--col": colors.get(leftId),
                    animationDelay: `${index * 60}ms`,
                  } as CSSProperties
                }
              />
            </span>
            <span className="an-bfl">{t(row.metric.label)}</span>
            <span className="an-bfb r">
              <i
                className={cn(!row.leftWins && "w")}
                style={
                  {
                    width: `${(row.right / row.max) * 100}%`,
                    "--col": colors.get(rightId),
                    animationDelay: `${index * 60}ms`,
                  } as CSSProperties
                }
              />
            </span>
            <span className={cn("an-bfv mono", !row.leftWins && "w")}>
              {display(t, row.metric, row.right)}
            </span>
          </div>
        ))}
      </div>
      <footer className="an-bff">
        <span>{rt("<b>{0}</b> is better on {1} of {2}", BOLD, left.label, wins, rows.length)}</span>
        {order ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!canRoute || reorder.isPending}
            title={canRoute ? undefined : t("Changing routing needs the right to edit providers")}
            onClick={() => reorder.mutate(order)}
          >
            <Ic n="swap" s={12} />
            {t("Put {0} first", left.label)}
          </Button>
        ) : (
          routable && <span className="an-note">{t("{0} already goes first", left.label)}</span>
        )}
      </footer>
    </AnalyticsCard>
  );
});

function ProviderMark({ provider }: { provider: AIAnalyticsProvider | undefined }) {
  if (!provider) {
    return null;
  }

  return (
    <Mark
      provider={{
        name: provider.name,
        kind: provider.kind ?? undefined,
        baseUrl: provider.baseUrl,
      }}
      s={22}
    />
  );
}

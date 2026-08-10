import { CarrierAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import {
  getRoutingGuideOptionsGraphQL,
  matchRoutingGuideGraphQL,
  type MatchedRoutingGuide,
  type RoutingGuideOption,
} from "@/lib/graphql/routing-guide-table";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { carrierRateMethodChoices, offerTtlChoices, tenderChannelChoices } from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  formatRoutingGuideLane,
  ROUTING_GUIDE_TIER_LABEL,
} from "@trenova/shared/types/routing-guide";
import {
  emptySpotTenderLine,
  spotTenderPayloadSchema,
  TENDER_CHANNEL_LABEL,
  type SpotTenderMode,
  type SpotTenderPayload,
  type SpotTenderPayloadInput,
} from "@trenova/shared/types/tender";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { TenderLivePanel } from "./tender-live-panel";
import type { DispatchActions } from "./use-dispatch-actions";

type GuidePreview = MatchedRoutingGuide | RoutingGuideOption;

const SPOT_MODE_ITEMS: { value: SpotTenderMode; label: string; caption: string }[] = [
  {
    value: "SpotBroadcast",
    label: "Broadcast",
    caption: "All carriers at once — first accept wins",
  },
  {
    value: "SpotSequential",
    label: "Sequential",
    caption: "One at a time, in the order listed",
  },
];

function entryRate(rate: string, rateMethod: string): string {
  const amount = Number(rate);
  if (!Number.isFinite(amount)) return rate;
  return rateMethod === "PerMile" ? `${formatCurrency(amount)}/mi` : formatCurrency(amount);
}

function GuideEntriesPreview({ guide }: { guide: GuidePreview }) {
  const entries = [...(guide.entries ?? [])].sort((a, b) => a.rank - b.rank);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-xs font-medium">{guide.name}</span>
        <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
          {ROUTING_GUIDE_TIER_LABEL[guide.specificity] ?? "Unranked"}
        </Badge>
        <span className="text-[10px] text-muted-foreground">{formatRoutingGuideLane(guide)}</span>
      </div>
      {entries.map((entry) => (
        <div
          key={entry.id}
          className="flex items-center justify-between gap-2 rounded border bg-muted/30 px-2 py-1"
        >
          <div className="flex min-w-0 items-center gap-1.5">
            <Badge variant="outline" className="h-4 shrink-0 rounded px-1 text-[9px] tabular-nums">
              #{entry.rank}
            </Badge>
            <span className="truncate text-[11px] font-medium">
              {entry.carrier?.name ?? "Unknown carrier"}
            </span>
          </div>
          <span className="shrink-0 text-[10px] text-muted-foreground tabular-nums">
            {entryRate(entry.rate, entry.rateMethod)} ·{" "}
            {formatDurationFromSeconds(entry.offerTtlSeconds)} ·{" "}
            {TENDER_CHANNEL_LABEL[entry.channel]}
          </span>
        </div>
      ))}
      {entries.length === 0 && (
        <p className="py-2 text-center text-[11px] text-muted-foreground">
          This guide has no carrier entries.
        </p>
      )}
    </div>
  );
}

function WaterfallTab({ move, actions }: { move: DispatchBoardMove; actions: DispatchActions }) {
  const [overrideGuideId, setOverrideGuideId] = useState("");

  const matchInput = useMemo(
    () => ({
      originLocationId: move.originLocationId ?? undefined,
      destinationLocationId: move.destinationLocationId ?? undefined,
      originCity: move.originCity || undefined,
      originState: move.originState || undefined,
      destinationCity: move.destinationCity || undefined,
      destinationState: move.destinationState || undefined,
    }),
    [move],
  );

  const { data: matchedGuide, isLoading: isMatching } = useQuery({
    queryKey: ["dispatch-tender-guide-match", matchInput] as const,
    queryFn: () => matchRoutingGuideGraphQL(matchInput),
  });

  const { data: guideOptions } = useQuery({
    queryKey: ["dispatch-tender-guide-options"] as const,
    queryFn: () => getRoutingGuideOptionsGraphQL(""),
  });

  const overrideGuide = overrideGuideId
    ? (guideOptions?.find((guide) => guide.id === overrideGuideId) ?? null)
    : null;
  const selectedGuide: GuidePreview | null = overrideGuide ?? matchedGuide ?? null;
  const usingOverride = Boolean(overrideGuide);

  const startWaterfall = useCallback(() => {
    void actions
      .startWaterfallTender({
        shipmentMoveId: move.moveId,
        routingGuideId: overrideGuideId || null,
      })
      .catch(() => {
        // handled by the mutation's onError; the dialog stays open for a retry
      });
  }, [actions, move.moveId, overrideGuideId]);

  return (
    <div className="flex flex-col gap-3">
      {isMatching ? (
        <Skeleton className="h-24 rounded-md" />
      ) : selectedGuide ? (
        <>
          {!usingOverride && (
            <p className="text-[11px] text-muted-foreground">
              Matched from this move&apos;s lane. Offers go out to each carrier in rank order until
              one accepts.
            </p>
          )}
          <GuideEntriesPreview guide={selectedGuide} />
        </>
      ) : (
        <p className="py-2 text-[11px] text-muted-foreground">
          No routing guide matches this lane. Pick one explicitly below, or{" "}
          <Link to="/dispatch/routing-guides" className="underline">
            create a routing guide
          </Link>{" "}
          for it.
        </p>
      )}

      <div className="flex flex-col gap-1">
        <span className="text-[10px] tracking-wide text-muted-foreground uppercase">
          Override guide
        </span>
        <Select
          value={overrideGuideId}
          onValueChange={(value) => setOverrideGuideId(!value || value === "auto" ? "" : value)}
        >
          <SelectTrigger className="h-8 text-xs">
            <SelectValue placeholder="Use the matched guide" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="auto">Use the matched guide</SelectItem>
            {(guideOptions ?? []).map((guide) => (
              <SelectItem key={guide.id} value={guide.id}>
                {guide.name} · {formatRoutingGuideLane(guide)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex justify-end">
        <Button
          type="button"
          disabled={!selectedGuide || (selectedGuide.entries?.length ?? 0) === 0}
          isLoading={actions.isTendering}
          loadingText="Starting..."
          onClick={startWaterfall}
        >
          Start waterfall
        </Button>
      </div>
    </div>
  );
}

function SpotTab({ move, actions }: { move: DispatchBoardMove; actions: DispatchActions }) {
  const form = useForm<SpotTenderPayloadInput, unknown, SpotTenderPayload>({
    resolver: zodResolver(spotTenderPayloadSchema),
    defaultValues: {
      shipmentMoveId: move.moveId,
      mode: "SpotBroadcast",
      lines: [{ ...emptySpotTenderLine }],
    },
  });
  const { control, handleSubmit, setValue } = form;
  const { fields, append, remove } = useFieldArray({ control, name: "lines" });
  const mode = useWatch({ control, name: "mode" });
  const lines = useWatch({ control, name: "lines" }) ?? [];

  const onSubmit = useCallback(
    async (values: SpotTenderPayload) => {
      try {
        await actions.sendSpotTender(values);
      } catch {
        // handled by the mutation's onError; the form stays intact for a retry
      }
    },
    [actions],
  );

  return (
    <Form
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit(onSubmit)(event);
      }}
    >
      <div className="flex flex-col gap-3">
        <SegmentedControl<SpotTenderMode>
          items={SPOT_MODE_ITEMS}
          value={(mode as SpotTenderMode) ?? "SpotBroadcast"}
          onValueChange={(value) => setValue("mode", value, { shouldDirty: true })}
          fullWidth
          aria-label="Spot tender mode"
        />

        {fields.map((field, index) => (
          <div key={field.id} className="rounded-md border p-3">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-xs font-medium">Carrier {index + 1}</p>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="h-6 text-xs"
                disabled={fields.length === 1}
                onClick={() => remove(index)}
              >
                Remove
              </Button>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              <FormControl>
                <CarrierAutocompleteField
                  control={control}
                  name={`lines.${index}.carrierId`}
                  label="Carrier"
                  placeholder="Select carrier"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name={`lines.${index}.rateMethod`}
                  label="Rate Method"
                  placeholder="Select method"
                  rules={{ required: true }}
                  options={carrierRateMethodChoices}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`lines.${index}.rate`}
                  label="Rate"
                  placeholder="1500.00"
                  sideText="$"
                  decimalScale={2}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name={`lines.${index}.offerTtlSeconds`}
                  label="Offer Expiry"
                  placeholder="Select expiry"
                  rules={{ required: true }}
                  options={offerTtlChoices}
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name={`lines.${index}.channel`}
                  label="Channel"
                  placeholder="Select channel"
                  rules={{ required: true }}
                  options={tenderChannelChoices}
                />
              </FormControl>
              {lines[index]?.channel !== "EDI" && (
                <FormControl>
                  <InputField
                    control={control}
                    name={`lines.${index}.email`}
                    label="Email Override"
                    placeholder="dispatch@carrier.com"
                    description="Optional — defaults to the carrier's tender contact."
                  />
                </FormControl>
              )}
            </div>
          </div>
        ))}

        <div className="flex items-center justify-between">
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => append({ ...emptySpotTenderLine })}
          >
            Add carrier
          </Button>
          <Button type="submit" isLoading={actions.isTendering} loadingText="Sending...">
            Send offers
          </Button>
        </div>
      </div>
    </Form>
  );
}

/**
 * Tendering entry point for the console. If a live tender already rides on the
 * move the dialog shows its progress; otherwise it offers the two ways to
 * start one — a routing-guide waterfall or an ad-hoc spot tender.
 */
export function TenderDialog({
  move,
  actions,
  onClose,
}: {
  move: DispatchBoardMove;
  actions: DispatchActions;
  onClose: () => void;
}) {
  const { data: liveTender, isLoading } = useQuery({
    ...dispatchConsoleQueries.liveTender(move.moveId),
  });

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[680px]">
        <DialogHeader>
          <DialogTitle>{liveTender ? "Live Tender" : "Tender to Carriers"}</DialogTitle>
          <DialogDescription>
            {move.proNumber} · {move.originCity}, {move.originState} → {move.destinationCity},{" "}
            {move.destinationState}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[65vh] pr-2">
          {isLoading ? (
            <div className="flex flex-col gap-2 py-1">
              <Skeleton className="h-6 w-40 rounded" />
              <Skeleton className="h-20 rounded-md" />
              <Skeleton className="h-20 rounded-md" />
            </div>
          ) : liveTender ? (
            <TenderLivePanel
              tender={liveTender}
              isTendering={actions.isTendering}
              onCancelTender={actions.cancelTender}
              onRecordResponse={actions.recordOfferResponse}
            />
          ) : (
            <Tabs defaultValue="waterfall">
              <TabsList className="mb-3">
                <TabsTrigger value="waterfall">Waterfall</TabsTrigger>
                <TabsTrigger value="spot">Spot</TabsTrigger>
              </TabsList>
              <TabsContent value="waterfall">
                <WaterfallTab move={move} actions={actions} />
              </TabsContent>
              <TabsContent value="spot">
                <SpotTab move={move} actions={actions} />
              </TabsContent>
            </Tabs>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}

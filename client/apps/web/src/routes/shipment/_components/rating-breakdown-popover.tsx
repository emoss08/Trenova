import { BillReceipt } from "@/components/tariff-bill/bill-receipt";
import { ChoicesView } from "@/components/tariff-bill/choices-view";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { ArrowUpRightIcon, ChevronRightIcon, ReceiptIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type { Shipment } from "@trenova/shared/types/shipment";
import {
  tariffInputValuesSchema,
  type TariffBill,
  type TariffChoice,
} from "@trenova/shared/types/tariff";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { RatingContractReasoning } from "./rating-contract-reasoning";

/**
 * One quiet row under the receipt that opens to more: its label, a hint of what is inside, and
 * the content, which is mounted only while the row is open.
 */
function BreakdownDisclosure({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: ReactNode;
  children: ReactNode;
}) {
  return (
    <Collapsible className="border-t">
      <CollapsibleTrigger className="group hover:bg-surface-hover ui-focus-ring flex w-full items-center gap-2 px-3 py-2 text-left text-xs">
        <ChevronRightIcon className="text-foreground-subtle size-3.5 shrink-0 transition-transform group-data-[panel-open]:rotate-90" />
        <span className="font-medium">{label}</span>
        <span className="flex-1" />
        {hint ? <span className="text-foreground-muted min-w-0 truncate">{hint}</span> : null}
      </CollapsibleTrigger>
      <CollapsibleContent className="px-3 pt-1 pb-3">{children}</CollapsibleContent>
    </Collapsible>
  );
}

/**
 * Everything behind the charge summary's last rating, opened from the icon beside its title. It
 * reads as a receipt: the tariff version and when it ran, the freight and the accessorials each
 * under their subtotal, one total, and what the tariff noted or asked to review. What explains it
 * waits in rows beneath: what each `best` or `worst` chose, the input values it ran with and,
 * when a contract priced it, why that contract won, read from the server only when opened.
 */
export function RatingBreakdownPopover({
  shipmentId,
  ratingDetail,
  bill,
  choices,
}: {
  shipmentId: string | null | undefined;
  ratingDetail: NonNullable<Shipment["ratingDetail"]>;
  bill: TariffBill;
  choices: TariffChoice[];
}) {
  const t = useT();
  const currency = bill.currency || undefined;
  const skipped = ratingDetail.skipped ?? [];
  // A shipment read over GraphQL reaches the form unparsed, so the stored inputs are read here.
  const parsedInputs = tariffInputValuesSchema.safeParse(ratingDetail.inputs ?? {});
  const inputs = Object.entries(parsedInputs.success ? parsedInputs.data : {});

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button
                  type="button"
                  variant="quiet"
                  size="icon-xs"
                  aria-label={t("Rating breakdown")}
                  className="cursor-help"
                >
                  <ReceiptIcon className="size-3" />
                </Button>
              }
            />
          }
        />
        <TooltipContent>
          {ratingDetail.rateQuoteId
            ? t("See each charge, the inputs it ran with and why this contract won")
            : t("See each charge and the inputs it ran with")}
        </TooltipContent>
      </Tooltip>
      <PopoverContent align="start" className="w-96 gap-0 overflow-hidden p-0">
        <PopoverHeader className="gap-1 border-b px-3 py-2.5">
          <div className="flex items-center justify-between gap-3">
            <PopoverTitle className="text-sm">{t("Last rating")}</PopoverTitle>
            {ratingDetail.formulaTemplateId ? (
              <Link
                to={formulaTemplateRoutes.edit(ratingDetail.formulaTemplateId)}
                className="text-foreground-muted hover:text-foreground ui-focus-ring flex items-center gap-0.5 rounded-sm text-xs"
              >
                {t("Open tariff")}
                <ArrowUpRightIcon className="size-3" />
              </Link>
            ) : null}
          </div>
          <PopoverDescription className="text-foreground-muted flex min-w-0 items-center gap-1.5 text-xs">
            <span className="truncate">{ratingDetail.formulaTemplateName}</span>
            {ratingDetail.versionNumber ? (
              <span className="shrink-0 font-mono">{t("v{0}", ratingDetail.versionNumber)}</span>
            ) : null}
            <span aria-hidden className="shrink-0">
              ·
            </span>
            <span className="shrink-0">{formatUnixDateTimeMedium(ratingDetail.ratedAt)}</span>
          </PopoverDescription>
        </PopoverHeader>
        <div className="max-h-128 overflow-y-auto">
          <div className="flex flex-col gap-3 p-3">
            <BillReceipt bill={bill} currency={currency} />
            {skipped.length > 0 && (
              <p className="text-2xs text-foreground-subtle">
                {t(
                  "{0, plural, one {Not added again, since the tariff billed it:} other {Not added again, since the tariff billed them:}} {1}",
                  skipped.length,
                  skipped.map((skip) => skip.code).join(", "),
                )}
              </p>
            )}
          </div>
          {choices.length > 0 && (
            <BreakdownDisclosure
              label={t("Choices")}
              hint={t("{0, plural, one {# choice} other {# choices}}", choices.length)}
            >
              <ChoicesView choices={choices} currency={currency} />
            </BreakdownDisclosure>
          )}
          {inputs.length > 0 && (
            <BreakdownDisclosure
              label={t("Inputs it ran with")}
              hint={t("{0, plural, one {# input} other {# inputs}}", inputs.length)}
            >
              <dl aria-label={t("Inputs it ran with")} className="flex flex-col gap-1">
                {inputs.map(([name, value]) => (
                  <div key={name} className="flex items-baseline justify-between gap-4 text-xs">
                    <dt className="text-foreground-muted truncate">{humanizeTariffName(name)}</dt>
                    <dd className="font-mono tabular-nums">
                      {formatTariffValue(value, t, { currency })}
                    </dd>
                  </div>
                ))}
              </dl>
            </BreakdownDisclosure>
          )}
          {shipmentId && ratingDetail.rateQuoteId ? (
            <BreakdownDisclosure
              label={t("Why this contract")}
              hint={ratingDetail.agreementName || undefined}
            >
              <RatingContractReasoning shipmentId={shipmentId} />
            </BreakdownDisclosure>
          ) : null}
        </div>
      </PopoverContent>
    </Popover>
  );
}

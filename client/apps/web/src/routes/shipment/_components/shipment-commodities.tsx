import { CommodityAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { NumberField } from "@/components/fields/number-field";
import { recordPath } from "@/config/record-links";
import { fieldErrorMessages } from "@/lib/form-errors";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { shipment } from "@/lib/queries/shipment";
import { commodityFromOption, commodityTotals } from "@/lib/shipment-commodity";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { CapabilityExplainer } from "@trenova/shared/components/capability-explainer";
import {
  CapabilityFields,
  type FieldDescriptor,
} from "@trenova/shared/components/capability-form-section";
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  BiohazardIcon,
  BoxesIcon,
  Edit02Icon,
  PlusIcon,
  TrailerIcon,
  Trash01Icon,
  Truck01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import {
  CAPABILITIES,
  getProfile,
  isCapabilitySectionVisible,
} from "@trenova/shared/lib/capability";
import { describeCommodityDimensions } from "@trenova/shared/lib/permit";
import { cn, findDuplicateIds } from "@trenova/shared/lib/utils";
import type {
  ResolvedModeProfile,
  Shipment,
  ShipmentCommodity,
} from "@trenova/shared/types/shipment";
import { useState } from "react";
import {
  useFieldArray,
  useFormContext,
  useFormState,
  useWatch,
  type Path,
  type UseFieldArrayUpdate,
} from "react-hook-form";
import { Link } from "react-router";

/** Which line the dialog edits, and whether it was just added (so cancelling removes it). */
type DialogTarget = { index: number; mode: "add" | "edit" };

/** The table's column classes, with the dimensions column only when the mode shows it. */
function columnLayout(showDimensions: boolean) {
  return showDimensions
    ? { grid: "grid-cols-12", name: "col-span-3" }
    : { grid: "grid-cols-10", name: "col-span-4" };
}

function CommodityDialog({
  target,
  onClose,
  maxShipmentWeightLimit,
  checkHazmatSegregation,
  profile,
  update,
}: {
  target: DialogTarget;
  /** Closes the dialog; `saved` is false when the person cancelled. */
  onClose: (saved: boolean) => void;
  maxShipmentWeightLimit?: number;
  checkHazmatSegregation?: boolean;
  profile: ResolvedModeProfile | null;
  update: UseFieldArrayUpdate<Shipment, "commodities">;
}) {
  const t = useT();
  const { index } = target;
  const { control, setValue, getValues, setError, clearErrors } = useFormContext<Shipment>();
  const [saving, setSaving] = useState(false);
  const [checkFailed, setCheckFailed] = useState(false);
  const showDimensions = isCapabilitySectionVisible(profile, CAPABILITIES.dimensionalCargo);

  // The three sides share one rule, one capability gate and one explainer, so they are described
  // once and rendered by the shared renderer. The mode profile's dimension rule names
  // `commodities`, not the sides, so that is what resolves the required state.
  const sides = [
    ["lengthFeet", t("Length")],
    ["widthFeet", t("Width")],
    ["heightFeet", t("Height")],
  ] as const;
  const dimensionDescriptors: FieldDescriptor[] = sides.map(([field, label]) => ({
    name: `commodities.${index}.${field}`,
    ruleField: "commodities",
    capability: CAPABILITIES.dimensionalCargo,
    hideExplainer: true,
    render: ({ required }) => (
      <NumberField
        control={control}
        name={`commodities.${index}.${field}`}
        label={label}
        placeholder="0"
        sideText="ft"
        rules={{ required }}
      />
    ),
  }));

  function handleCommoditySelected(option: GraphQLSelectOption | null) {
    setValue(`commodities.${index}.commodity`, option ? commodityFromOption(option) : null);
  }

  async function handleSave() {
    const values = getValues(`commodities.${index}`);
    const commodities = getValues("commodities") ?? [];
    const others = commodities.filter((_, at) => at !== index);
    const nextWeight = typeof values.weight === "number" ? values.weight : 0;

    if (
      typeof maxShipmentWeightLimit === "number" &&
      commodityTotals(others).weight + nextWeight > maxShipmentWeightLimit
    ) {
      setError(`commodities.${index}.weight`, {
        type: "manual",
        message: t(
          "Total commodity weight cannot exceed {0} lbs",
          formatNumber(maxShipmentWeightLimit),
        ),
      });
      return;
    }
    clearErrors(`commodities.${index}.weight`);
    setCheckFailed(false);

    const commodityIds = commodities
      .map((line, at) => (at === index ? values.commodityId : line?.commodityId))
      .filter((id): id is string => !!id);

    if (checkHazmatSegregation && commodityIds.length >= 2) {
      setSaving(true);
      try {
        await apiService.shipmentService.checkHazmatSegregation(commodityIds);
        commodities.forEach((_, at) => clearErrors(`commodities.${at}.commodityId`));
      } catch (error) {
        if (error instanceof ApiRequestError && error.isValidationError()) {
          // The server names the lines that cannot travel together by their form paths.
          for (const fieldError of error.getFieldErrors()) {
            setError(fieldError.field as Path<Shipment>, {
              type: "manual",
              message: fieldError.message,
            });
          }
          update(index, values);
          return;
        }
        // Without the check the lines cannot be cleared to travel together, so nothing is saved.
        setCheckFailed(true);
        return;
      } finally {
        setSaving(false);
      }
    }

    update(index, values);
    onClose(true);
  }

  const editing = target.mode === "edit";

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose(false);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? t("Edit commodity") : t("Add commodity")}</DialogTitle>
          <DialogDescription>
            {editing
              ? t("Update the commodity details")
              : t("Select a commodity and specify quantity and weight")}
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={1}>
          <FormControl>
            <CommodityAutocompleteField
              control={control}
              name={`commodities.${index}.commodityId`}
              label={t("Commodity")}
              placeholder={t("Select commodity")}
              onOptionChange={handleCommoditySelected}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`commodities.${index}.pieces`}
              label={t("Pieces")}
              placeholder="1"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`commodities.${index}.weight`}
              label={t("Weight (lbs)")}
              placeholder="0"
            />
          </FormControl>
        </FormGroup>
        <CapabilityFields descriptors={dimensionDescriptors} profile={profile} cols={3} />
        {showDimensions && <CapabilityExplainer profile={profile} field="commodities" />}
        {checkFailed && (
          <Alert variant="destructive" size="sm">
            <AlertTriangleIcon />
            <AlertDescription>
              {t(
                "The hazardous material check could not run, so the commodity was not saved. Try again.",
              )}
            </AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onClose(false)} disabled={saving}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            onClick={() => void handleSave()}
            isLoading={saving}
            loadingText={t("Checking…")}
          >
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** A handling flag beside a line's name, explained on hover or focus. */
function CommodityFlag({
  icon: Icon,
  className,
  label,
}: {
  icon: IconComponent;
  className: string;
  label: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            role="img"
            tabIndex={0}
            aria-label={label}
            className="ui-focus-ring inline-flex cursor-help rounded-sm"
          />
        }
      >
        <Icon className={cn("size-3.5", className)} />
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

function CommodityRow({
  line,
  errors,
  duplicate,
  showDimensions,
  onEdit,
  onRemove,
}: {
  line: ShipmentCommodity | undefined;
  /** The line's errors as the form holds them, nested ones included. */
  errors: object | undefined;
  duplicate: boolean;
  showDimensions: boolean;
  onEdit: () => void;
  onRemove: () => void;
}) {
  const t = useT();
  const layout = columnLayout(showDimensions);
  const commodity = line?.commodity;
  const name = commodity?.name || "—";
  const messages = errors ? fieldErrorMessages(errors) : [];
  const invalid = messages.length > 0;
  const flags = [
    commodity?.hazardousMaterialId
      ? {
          key: "hazmat",
          icon: BiohazardIcon,
          className: "text-warning",
          label: t("Commodity is classified as hazardous material."),
        }
      : null,
    commodity?.stackable
      ? {
          key: "stackable",
          icon: BoxesIcon,
          className: "text-success",
          label: t("Commodity is marked as stackable."),
        }
      : null,
    commodity?.fragile
      ? {
          key: "fragile",
          icon: AlertCircleIcon,
          className: "text-destructive",
          label: t("Commodity is marked as fragile."),
        }
      : null,
  ].filter((flag) => flag !== null);

  return (
    <div
      className={cn(
        "grid items-center gap-2 px-4 py-2",
        layout.grid,
        invalid && "bg-danger-subtle ring-destructive ring-1 ring-inset",
        !invalid && duplicate && "bg-warning-subtle ring-warning ring-1 ring-inset",
      )}
    >
      <div className={cn("flex min-w-0 items-center gap-1.5", layout.name)}>
        {commodity?.id ? (
          <Link
            to={recordPath("commodity", commodity.id)}
            title={name}
            className="text-primary min-w-0 truncate text-xs font-medium hover:underline"
          >
            {name}
          </Link>
        ) : (
          <span title={name} className="min-w-0 truncate text-xs font-medium">
            {name}
          </span>
        )}
        {flags.length > 0 && (
          <span className="flex shrink-0 items-center gap-1">
            {flags.map(({ key, ...flag }) => (
              <CommodityFlag key={key} {...flag} />
            ))}
          </span>
        )}
      </div>
      <span className="text-muted-foreground col-span-2 truncate text-xs tabular-nums">
        {formatNumber(line?.pieces ?? 0)}
      </span>
      <span className="text-muted-foreground col-span-2 truncate text-xs tabular-nums">
        {t("{0} lbs", formatNumber(line?.weight ?? 0))}
      </span>
      {showDimensions && (
        <span className="text-muted-foreground col-span-3 truncate text-xs tabular-nums">
          {describeCommodityDimensions(line)}
        </span>
      )}
      <div className="col-span-2 flex items-center justify-end gap-1">
        {invalid && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span
                  role="img"
                  tabIndex={0}
                  aria-label={t("This commodity has problems")}
                  className="ui-focus-ring inline-flex cursor-help rounded-sm"
                />
              }
            >
              <AlertTriangleIcon className="text-destructive size-3.5" />
            </TooltipTrigger>
            <TooltipContent side="top">
              <ul className="space-y-1">
                {messages.map((message) => (
                  <li key={message} className="text-xs">
                    {message}
                  </li>
                ))}
              </ul>
            </TooltipContent>
          </Tooltip>
        )}
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Edit {0}", name)}
          onClick={onEdit}
        >
          <Edit02Icon className="text-muted-foreground size-3.5" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Remove {0}", name)}
          onClick={onRemove}
        >
          <Trash01Icon className="text-muted-foreground size-3.5" />
        </Button>
      </div>
    </div>
  );
}

/**
 * The shipment's commodity lines: each line's commodity with its handling flags, pieces, weight
 * and (when the mode tracks them) dimensions, the totals against the organisation's weight limit,
 * and the dialog that adds or edits a line, checking the weight limit and, when the organisation
 * asks for it, that the lines' hazardous materials may travel together.
 */
export default function CommoditiesSection() {
  const t = useT();

  const { control } = useFormContext<Shipment>();
  // Only the commodities' errors, so an error elsewhere in the shipment does not redraw the list.
  const { errors } = useFormState({ control, name: "commodities" });
  const { fields, append, update, remove } = useFieldArray({
    control,
    name: "commodities",
    keyName: "fieldId",
  });
  const commodities = useWatch({ control, name: "commodities" }) ?? [];
  const { data: shipmentUIPolicy } = useQuery({ ...shipment.uiPolicy() });
  const [target, setTarget] = useState<DialogTarget | null>(null);

  const profile = getProfile(shipmentUIPolicy);
  const showDimensions = isCapabilitySectionVisible(profile, CAPABILITIES.dimensionalCargo);
  const layout = columnLayout(showDimensions);
  const duplicateCommodityIds = findDuplicateIds(commodities, (line) => line?.commodityId);
  const totals = commodityTotals(commodities);
  const weightLimit = shipmentUIPolicy?.maxShipmentWeightLimit;

  function handleAdd() {
    append({ commodityId: "", pieces: 1, weight: 0 });
    setTarget({ index: fields.length, mode: "add" });
  }

  function handleClose(saved: boolean) {
    if (!saved && target?.mode === "add") remove(target.index);
    setTarget(null);
  }

  return (
    <>
      <FormSection
        title={t("Commodities")}
        titleCount={commodities.length}
        description={t("Cargo items, weights, and hazardous material compliance")}
        action={
          fields.length > 0 && (
            <Button type="button" variant="outline" size="xxs" onClick={handleAdd}>
              <PlusIcon className="size-3" />
              {t("Add commodity")}
            </Button>
          )
        }
      >
        {fields.length > 0 ? (
          <div className="border-border bg-card rounded-lg border">
            <div
              className={cn(
                "border-border text-muted-foreground grid gap-2 border-b px-4 py-2 text-xs",
                layout.grid,
              )}
            >
              <span className={layout.name}>{t("Commodity")}</span>
              <span className="col-span-2">{t("Pieces")}</span>
              <span className="col-span-2">{t("Weight")}</span>
              {showDimensions && <span className="col-span-3">{t("L × W × H")}</span>}
              <span className="col-span-2" />
            </div>
            <div className="divide-y">
              {fields.map((field, index) => {
                const line = commodities[index];
                return (
                  <CommodityRow
                    key={field.fieldId}
                    line={line}
                    errors={errors.commodities?.[index]}
                    duplicate={!!line?.commodityId && duplicateCommodityIds.has(line.commodityId)}
                    showDimensions={showDimensions}
                    onEdit={() => setTarget({ index, mode: "edit" })}
                    onRemove={() => remove(index)}
                  />
                );
              })}
            </div>
            <div className="border-border bg-muted flex items-center justify-end gap-3 rounded-b-lg border-t px-4 py-2 text-xs">
              <span className="text-muted-foreground">
                {t("{0, plural, one {# piece} other {# pieces}} total", totals.pieces)}
              </span>
              <span className="flex items-center gap-1 tabular-nums">
                <Tooltip>
                  <TooltipTrigger
                    render={<span tabIndex={0} className="ui-focus-ring cursor-help font-medium" />}
                  >
                    {t("{0} lbs", formatNumber(totals.weight))}
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    {t("Total weight of all commodities in the shipment (lbs).")}
                  </TooltipContent>
                </Tooltip>
                {typeof weightLimit === "number" && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span
                          tabIndex={0}
                          className="text-muted-foreground ui-focus-ring cursor-help"
                        />
                      }
                    >
                      {t("/ {0} lbs", formatNumber(weightLimit))}
                    </TooltipTrigger>
                    <TooltipContent side="top">
                      {t("Maximum shipment weight limit configured by organization.")}
                    </TooltipContent>
                  </Tooltip>
                )}
              </span>
            </div>
          </div>
        ) : (
          <EmptyState
            className="rounded-lg border p-4"
            title={t("No commodities")}
            description={t("Shipment has no associated commodities")}
            icons={[TrailerIcon, BoxesIcon, Truck01Icon]}
            action={{
              label: t("Add first commodity"),
              onClick: handleAdd,
              icon: PlusIcon,
            }}
          />
        )}
        {duplicateCommodityIds.size > 0 && (
          <p className="text-warning flex items-center gap-1 text-xs">
            <AlertTriangleIcon className="size-3.5" />
            {t("Duplicate commodities detected in this shipment.")}
          </p>
        )}
      </FormSection>
      {target && (
        <CommodityDialog
          target={target}
          onClose={handleClose}
          maxShipmentWeightLimit={weightLimit}
          checkHazmatSegregation={shipmentUIPolicy?.checkHazmatSegregation}
          profile={profile}
          update={update}
        />
      )}
    </>
  );
}

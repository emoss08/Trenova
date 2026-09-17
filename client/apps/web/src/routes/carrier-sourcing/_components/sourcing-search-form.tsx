import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import type { CarrierSourcingSuggestion } from "@/lib/graphql/carrier-sourcing";
import { usStateAbbreviationChoices } from "@/lib/choices";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { RotateCcwIcon, SearchIcon } from "lucide-react";
import { useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import {
  SOURCING_SEARCH_DEFAULTS,
  sourcingSearchSchema,
  type SourcingSearchFormValues,
} from "./sourcing-schema";
import { SourcingTextAutocomplete } from "./sourcing-text-autocomplete";

export type SourcingSearchFormProps = {
  initialValues: SourcingSearchFormValues | null;
  autocompleteEnabled: boolean;
  isSearching: boolean;
  onSearch: (values: SourcingSearchFormValues) => void;
  onReset: () => void;
  onPickSuggestion: (suggestion: CarrierSourcingSuggestion) => void;
};

export function SourcingSearchForm({
  initialValues,
  autocompleteEnabled,
  isSearching,
  onSearch,
  onReset,
  onPickSuggestion,
}: SourcingSearchFormProps) {
  const t = useT();
  const stateOptions = useMemo(() => [...usStateAbbreviationChoices], []);

  const form = useForm<SourcingSearchFormValues>({
    resolver: zodResolver(sourcingSearchSchema) as Resolver<SourcingSearchFormValues>,
    defaultValues: initialValues ?? SOURCING_SEARCH_DEFAULTS,
  });
  const { control, handleSubmit, reset } = form;

  return (
    <FormProvider {...form}>
      <Form
        className="bg-card flex flex-col gap-3 rounded-lg border p-4"
        aria-label={t("Carrier search")}
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void handleSubmit(onSearch)(event);
        }}
      >
        <FormGroup cols={4} className="gap-y-3">
          <FormControl cols="full">
            <SourcingTextAutocomplete
              control={control}
              autocompleteEnabled={autocompleteEnabled}
              onPickSuggestion={onPickSuggestion}
            />
          </FormControl>
          <FormControl>
            <SelectField<SourcingSearchFormValues>
              control={control}
              name="state"
              label={t("Based in")}
              description={t("The carrier's physical address state.")}
              options={stateOptions}
              placeholder={t("Any state")}
              isClearable
            />
          </FormControl>
          <FormControl>
            <SelectField<SourcingSearchFormValues>
              control={control}
              name="originState"
              label={t("Lane origin")}
              description={t("Ranks carriers that run loads out of this state.")}
              options={stateOptions}
              placeholder={t("Any origin")}
              isClearable
            />
          </FormControl>
          <FormControl>
            <SelectField<SourcingSearchFormValues>
              control={control}
              name="destinationState"
              label={t("Lane destination")}
              description={t("Ranks carriers that deliver into this state.")}
              options={stateOptions}
              placeholder={t("Any destination")}
              isClearable
            />
          </FormControl>
          <FormControl>
            <NumberField<SourcingSearchFormValues, "minAuthorityAgeDays">
              control={control}
              name="minAuthorityAgeDays"
              label={t("Minimum authority age")}
              description={t("Oldest active operating authority.")}
              placeholder={t("Any age")}
              sideText={t("days")}
              min={0}
            />
          </FormControl>
          <FormControl>
            <NumberField<SourcingSearchFormValues, "minPowerUnits">
              control={control}
              name="minPowerUnits"
              label={t("Minimum power units")}
              placeholder={t("No minimum")}
              min={0}
            />
          </FormControl>
          <FormControl>
            <NumberField<SourcingSearchFormValues, "maxPowerUnits">
              control={control}
              name="maxPowerUnits"
              label={t("Maximum power units")}
              placeholder={t("No maximum")}
              min={0}
            />
          </FormControl>
          <FormControl cols="full" className="min-h-0">
            <div className="grid grid-cols-1 gap-1 sm:grid-cols-3 lg:max-w-3xl">
              <SwitchField<SourcingSearchFormValues>
                control={control}
                name="hazmatOnly"
                label={t("Hazmat carriers")}
                position="left"
              />
              <SwitchField<SourcingSearchFormValues>
                control={control}
                name="excludeBlocking"
                label={t("Hide blocked")}
                position="left"
              />
              <SwitchField<SourcingSearchFormValues>
                control={control}
                name="excludeExistingCarriers"
                label={t("Hide carriers we have")}
                position="left"
              />
            </div>
          </FormControl>
        </FormGroup>
        <div className="flex items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              reset(SOURCING_SEARCH_DEFAULTS);
              onReset();
            }}
          >
            <RotateCcwIcon className="size-3.5" />
            {t("Reset")}
          </Button>
          <Button type="submit" size="sm" isLoading={isSearching}>
            <SearchIcon className="size-3.5" />
            {t("Search carriers")}
          </Button>
        </div>
      </Form>
    </FormProvider>
  );
}

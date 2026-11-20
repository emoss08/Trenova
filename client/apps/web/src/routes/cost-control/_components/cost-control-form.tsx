import { SettingsForm } from "@/components/settings/settings-form";
import { getCostingControlGraphQL, saveCostingControlGraphQL } from "@/lib/graphql/cost-control";
import { costControl } from "@/lib/queries/cost-control";
import { shipment } from "@/lib/queries/shipment";
import { costControlSchema, type CostControlFormValues } from "@/types/cost-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  costControlRefusal,
  costControlSave,
  prepareCostControl,
  toCostControlFormValues,
} from "../cost-control-submit";
import { useCostControlTabs, type CostControlTab } from "../cost-control-tabs";
import { ActualsTab } from "./actuals-tab";
import { BasisTab } from "./basis-tab";
import { RatesTab } from "./rates-tab";

/** Every shipment's cost estimate and margin, and the analytics built on them, read this model. */
const COST_MODEL_READERS = [
  costControl.resolvedProfile._def,
  shipment._def,
  ["analytics"],
] as const;

/**
 * Cost control: the cost-per-mile model behind shipment profitability, one form saved once,
 * split into what the estimate assumes, what each cost category charges, and how rates are
 * read from the ledger. GL actuals are there only for an organization with accounting.
 */
export default function CostControlForm() {
  const { data } = useSuspenseQuery({ ...costControl.get() });
  const tabs = useCostControlTabs();
  const saved = toCostControlFormValues(data);

  const save = async (values: CostControlFormValues): Promise<CostControlFormValues> => {
    const change = costControlSave(values, saved);
    if (change) {
      try {
        await saveCostingControlGraphQL(change.input);
      } catch (error) {
        throw costControlRefusal(error, change.categoryIndexes);
      }
    }
    return toCostControlFormValues(await getCostingControlGraphQL());
  };

  return (
    <SettingsForm<CostControlFormValues, CostControlTab>
      values={saved}
      schema={costControlSchema}
      queryKey={costControl.get._def}
      save={save}
      refresh={COST_MODEL_READERS}
      resourceName="Cost Control"
      prepare={prepareCostControl}
      tabs={tabs}
      content={(tab, { selectTab }) => {
        switch (tab) {
          case "basis":
            return <BasisTab />;
          case "rates":
            return <RatesTab />;
          case "actuals":
            return <ActualsTab onOpenRates={() => selectTab("rates")} />;
        }
      }}
    />
  );
}

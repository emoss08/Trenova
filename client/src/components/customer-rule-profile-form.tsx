import { useDocumentClass } from "@/hooks/useQueries";
import { BillingCycleChoices, type CustomerFormValues } from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { SelectInput } from "./common/fields/select-input";
import { FormControl, FormGroup } from "./ui/form";

export function CustomerRuleProfileForm({ open }: { open: boolean }) {
  const { control } = useFormContext<CustomerFormValues>();

  const {
    selectDocumentClassData,
    isError: isDocumentClassError,
    isLoading: isDocumentClassLoading,
  } = useDocumentClass(open);

  return (
    <FormGroup className="my-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <FormControl>
        <SelectInput
          control={control}
          rules={{ required: true }}
          name="ruleProfile.billingCycle"
          label="Billing Cycle"
          options={BillingCycleChoices}
          placeholder="Billing Cycle"
          description="Specify the frequency of which the customer will be billed."
        />
      </FormControl>
      <FormControl>
        <SelectInput
          name="ruleProfile.docClassIds"
          control={control}
          isMulti
          rules={{ required: true }}
          label="Document Classification"
          options={selectDocumentClassData}
          isFetchError={isDocumentClassError}
          isLoading={isDocumentClassLoading}
          placeholder="Select Document Classification"
          description="Select the state or region for the customer."
          hasPopoutWindow
          popoutLink="/billing/document-classes/"
          popoutLinkLabel="Document Classification"
        />
      </FormControl>
    </FormGroup>
  );
}

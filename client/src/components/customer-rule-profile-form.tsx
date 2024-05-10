import { InputField } from "@/components/common/fields/input";
import { useDocumentClass } from "@/hooks/useQueries";
import { type CustomerFormValues } from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { SelectInput } from "./common/fields/select-input";

export function CustomerRuleProfileForm({ open }: { open: boolean }) {
  const { control } = useFormContext<CustomerFormValues>();

  const {
    selectDocumentClassData,
    isError: isDocumentClassError,
    isLoading: isDocumentClassLoading,
  } = useDocumentClass(open);

  return (
    <>
      <div className="my-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="ruleProfile.name"
              label="Name"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Name"
              description="Specify the official name of the customer."
              maxLength={50}
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <SelectInput
              name="ruleProfile.documentClass"
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
              popoutLink="#" // TODO: Change once Document Classification is added.
              popoutLinkLabel="Document Classification"
            />
          </div>
        </div>
      </div>
    </>
  );
}

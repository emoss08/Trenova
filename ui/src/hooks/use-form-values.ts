import { useFormContext, useWatch } from "react-hook-form";

export const useFormValues = () => {
  const { getValues } = useFormContext();

  return {
    ...useWatch(),
    ...getValues(),
  };
};

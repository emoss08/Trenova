import { SelectField } from "@/components/fields/select-field";
import { fetchPayCodeOptions, type PayCodeOption } from "@/lib/graphql/driver-settlement";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import type { Control, FieldPath, FieldValues } from "react-hook-form";

export function usePayCodeOptions(direction?: "Earning" | "Deduction") {
  return useQuery({
    queryKey: ["pay-code-options", direction ?? "all"],
    queryFn: () => fetchPayCodeOptions(direction),
    staleTime: 60_000,
  });
}

export function payCodeLabel(option: PayCodeOption): string {
  return `${option.code} — ${option.name}`;
}

export function PayCodeSelectField<T extends FieldValues>({
  control,
  name,
  direction,
  label = "Pay Code",
  description,
  required = true,
}: {
  control: Control<T>;
  name: FieldPath<T>;
  direction?: "Earning" | "Deduction";
  label?: string;
  description: string;
  required?: boolean;
}) {
  const { data: options } = usePayCodeOptions(direction);

  const items = useMemo(
    () =>
      (options ?? []).map((option) => ({
        label: direction ? payCodeLabel(option) : `${payCodeLabel(option)} (${option.direction})`,
        value: option.id,
      })),
    [options, direction],
  );

  return (
    <SelectField
      control={control}
      name={name}
      label={label}
      options={items}
      rules={{ required }}
      description={description}
    />
  );
}

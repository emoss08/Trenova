import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type {
  TariffInputDeclaration,
  TariffInputValues,
  TariffWireValue,
} from "@trenova/shared/types/tariff";
import { useWatch } from "react-hook-form";

export type TariffInputsFieldsProps = {
  /** The tariff's declared inputs, in declaration order, as `tariff.check` derives them. */
  declarations: readonly TariffInputDeclaration[];
  /**
   * The react-hook-form path the values live under, as a map from input name to wire value:
   * `inputs`, `tariffInputs`, `rules.${index}.inputs`.
   */
  name: string;
  /**
   * Values an input inherits when this form leaves it unset (the agreement's, under a rule),
   * shown as the field's placeholder; the declared default is the fallback after these.
   */
  inherited?: TariffInputValues;
  /** The rating currency's ISO code, for money placeholders. */
  currency?: string | null;
  disabled?: boolean;
};

/**
 * A tariff's inputs as form fields under `name`: labels from the doc comments, unit-aware
 * money, rate and quantity fields, enums, optionals, records and list-of-record tables (weight
 * breaks), each value kept in wire form. This read-only rendering lists each input with the
 * value it would take; the editable fields replace it.
 */
export function TariffInputsFields({
  declarations,
  name,
  inherited,
  currency,
}: TariffInputsFieldsProps) {
  const t = useT();
  const values = useWatch({ name }) as TariffInputValues | undefined;

  if (declarations.length === 0) {
    return <p className="text-foreground-muted text-sm">{t("This tariff declares no inputs.")}</p>;
  }

  const resolved = (declaration: TariffInputDeclaration): TariffWireValue | undefined =>
    values?.[declaration.name] ?? inherited?.[declaration.name] ?? declaration.default;

  return (
    <DescriptionList layout="inline">
      {declarations.map((declaration) => {
        const value = resolved(declaration);
        return (
          <DescriptionItem
            key={declaration.name}
            label={declaration.doc ?? declaration.name}
            numeric
          >
            {value === undefined ? (
              <span className="text-danger-foreground">{t("Required")}</span>
            ) : (
              formatTariffValue(value, t, { currency })
            )}
          </DescriptionItem>
        );
      })}
    </DescriptionList>
  );
}

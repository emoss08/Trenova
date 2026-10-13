import { Badge } from "@trenova/shared/components/ui/badge";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplateFormValues,
  FormulaTemplateType,
} from "@trenova/shared/types/formula-template";
import { Truck01Icon } from "@trenova/shared/components/icons";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import { sectionDomId } from "../studio-context";

/**
 * The template's name and what it is for, written in place: a large
 * borderless name, a description that grows as it is written, then what kind
 * of charge it computes and the schema it reads.
 */
export function DetailsHero() {
  const t = useT();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const { errors } = useFormState({ control, name: ["name", "description"] });

  return (
    <section id={sectionDomId("details")} className="flex scroll-mt-6 flex-col gap-2 pb-2">
      <Controller
        name="name"
        control={control}
        render={({ field, fieldState }) => (
          <div className="flex flex-col gap-1">
            <Textarea
              bare
              ref={field.ref}
              name={field.name}
              value={field.value ?? ""}
              onBlur={field.onBlur}
              onChange={(event) => field.onChange(event.target.value.replace(/\n/g, " "))}
              onKeyDown={(event) => {
                if (event.key === "Enter") event.preventDefault();
              }}
              minRows={1}
              maxLength={100}
              aria-label={t("Template name")}
              aria-invalid={fieldState.invalid || undefined}
              placeholder={t("Untitled rating method")}
              className={cn(
                "placeholder:text-foreground-subtle rounded-md px-1.5 py-0.5 text-2xl font-medium md:text-2xl",
                fieldState.invalid && "bg-danger-subtle",
              )}
            />
            {fieldState.error?.message && (
              <p className="text-danger px-1.5 text-xs">{fieldState.error.message}</p>
            )}
          </div>
        )}
      />
      <Controller
        name="description"
        control={control}
        render={({ field }) => (
          <Textarea
            bare
            ref={field.ref}
            name={field.name}
            value={field.value ?? ""}
            onBlur={field.onBlur}
            onChange={field.onChange}
            minRows={1}
            aria-label={t("Description")}
            aria-invalid={errors.description ? true : undefined}
            placeholder={t(
              "When this method applies and how it prices. Reviewers and rate agreement owners read this.",
            )}
            className={cn(
              "text-foreground-muted placeholder:text-foreground-subtle rounded-md px-1.5 py-0.5 text-sm leading-relaxed md:text-sm",
              errors.description && "bg-danger-subtle",
            )}
          />
        )}
      />
      <div className="flex flex-wrap items-center gap-2 px-1.5 pt-1">
        <span className="text-foreground-subtle text-xs">{t("Charges")}</span>
        <Controller
          name="type"
          control={control}
          render={({ field }) => (
            <SegmentedControl<FormulaTemplateType>
              aria-label={t("Charges")}
              value={field.value}
              onValueChange={field.onChange}
              items={[
                { value: "FreightCharge", label: t("Freight") },
                { value: "AccessorialCharge", label: t("Accessorial") },
              ]}
            />
          )}
        />
        <span className="text-foreground-subtle ml-2 text-xs">{t("Schema")}</span>
        <Badge variant="neutral" className="gap-1">
          <Truck01Icon className="size-3" />
          {t("Shipment")}
        </Badge>
      </div>
    </section>
  );
}

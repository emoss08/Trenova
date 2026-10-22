import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { InfoCircleIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  FORMULA_LIBRARY_NAME_MAX,
  type FormulaLibraryFormValues,
} from "@trenova/shared/types/formula-library";
import { Controller, useFormContext, useFormState } from "react-hook-form";
import { useLibraryStudio } from "./library-studio-context";
import { libraryPath } from "./starter";

/**
 * The library's name, which is the path templates import it by (`org.<name>`), and what it is
 * for. Renaming a library other templates import is called out: their source still says the
 * old path.
 */
export function LibraryDetails() {
  const t = useT();
  const { control } = useFormContext<FormulaLibraryFormValues>();
  const { errors } = useFormState({ control, name: ["name", "description"] });
  const { library, renamed } = useLibraryStudio();
  const importers = library?.usageCount ?? 0;

  return (
    <section className="flex flex-col gap-2 pb-2">
      <Controller
        name="name"
        control={control}
        render={({ field, fieldState }) => (
          <div className="flex flex-col gap-1">
            <Input
              ref={field.ref}
              name={field.name}
              value={field.value ?? ""}
              onBlur={field.onBlur}
              onChange={(event) => field.onChange(event.target.value.replace(/\s/g, ""))}
              maxLength={FORMULA_LIBRARY_NAME_MAX}
              spellCheck={false}
              autoComplete="off"
              aria-label={t("Library name")}
              aria-describedby="library-name-hint"
              aria-invalid={fieldState.invalid || undefined}
              placeholder={t("accessorials")}
              leftElement={<span className="text-foreground-subtle font-mono text-lg">org.</span>}
              className="h-10 pl-12 font-mono text-lg md:text-lg"
            />
            {fieldState.error?.message ? (
              <p id="library-name-hint" className="text-danger px-1.5 text-xs">
                {fieldState.error.message}
              </p>
            ) : (
              <p id="library-name-hint" className="text-foreground-muted px-1.5 text-xs">
                {t("Templates import it with use {0}", libraryPath(field.value || "name"))}
              </p>
            )}
          </div>
        )}
      />
      {renamed && importers > 0 && library && (
        <Alert variant="warning" size="sm">
          <InfoCircleIcon />
          <AlertDescription>
            {t(
              "{0, plural, one {# template version imports} other {# template versions import}} this library as {1}. They keep the version they pinned, but a new version of them must import the new name.",
              importers,
              libraryPath(library.name),
            )}
          </AlertDescription>
        </Alert>
      )}
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
              "What the library holds and when to import it. Reviewers and template authors read this.",
            )}
            className={cn(
              "text-foreground-muted placeholder:text-foreground-subtle rounded-md px-1.5 py-0.5 text-sm leading-relaxed md:text-sm",
              errors.description && "bg-danger-subtle",
            )}
          />
        )}
      />
    </section>
  );
}

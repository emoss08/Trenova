import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { TariffInputsFieldsSkeleton } from "@/components/tariff-inputs/tariff-inputs-fields-skeleton";
import { useTemplatesInputs } from "@/hooks/use-template-inputs";
import { agreementSharedInputs, rateAgreementTemplateIds } from "@/lib/rate-agreement-inputs";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The agreement's contract-wide terms: the inputs every lane's tariff declares alike, set once
 * here and read by each lane that leaves them unset (a minimum, a fuel peg). Only inputs all
 * the lanes share can be set at this level, because the server checks an agreement's value
 * against every lane's tariff.
 */
export function AgreementTariffInputs() {
  const t = useT();
  const { control } = useFormContext<RateAgreement>();
  const rules = useWatch({ control, name: "rules" });
  const currency = useWatch({ control, name: "currency" });
  const templateIds = rateAgreementTemplateIds(rules);
  const loaded = useTemplatesInputs(templateIds);

  if (templateIds.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t(
          "Add a lane with a tariff first. The terms its tariff declares can then be set here for every lane at once.",
        )}
      </p>
    );
  }

  if (loaded.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertTriangleIcon className="size-4" />
        <AlertDescription>{t("A lane's tariff inputs could not be read.")}</AlertDescription>
      </Alert>
    );
  }

  const shared = agreementSharedInputs(rules, loaded.byTemplate);
  if (!shared.complete) {
    return <TariffInputsFieldsSkeleton fields={2} />;
  }

  if (shared.declarations.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("The lanes' tariffs share no inputs, so each lane sets its own terms on the Lanes tab.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <p className="text-muted-foreground text-xs">
        {t(
          "{0, plural, one {Every lane's tariff reads these.} other {All # tariffs on this agreement's lanes read these.}} A lane that sets its own value keeps it.",
          templateIds.length,
        )}
      </p>
      <TariffInputsFields
        declarations={shared.declarations}
        name="inputs"
        currency={currency}
        requireUnset={false}
      />
    </div>
  );
}

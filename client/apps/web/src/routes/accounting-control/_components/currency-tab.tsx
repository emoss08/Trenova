import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { SettingRow, settingRowDescriptionClass } from "@/components/fields/field-components";
import { SelectField } from "@/components/fields/select-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { usePermissions } from "@/hooks/use-permission";
import {
  currencyChoices,
  currencyModeChoices,
  exchangeRateDatePolicyChoices,
  exchangeRateOverridePolicyChoices,
} from "@/lib/choices";
import { integration } from "@/lib/queries/integration";
import { OANDAExchangeRatesIntegrationModal } from "@/routes/admin/integrations/_components/oanda/oanda-integration-modal";
import { currencyModeSchema, type AccountingControl } from "@/types/accounting-control";
import { useQuery } from "@tanstack/react-query";
import { Sliders04Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const OANDA_INTEGRATION_TYPE = "OANDAExchangeRates";

/**
 * The currency the books are kept in and, for an organization that deals in several, where
 * exchange rates come from and where realized gains and losses post. Multi-currency needs
 * OANDA connected for its rates; its policies appear once it is.
 */
export function CurrencyTab() {
  const t = useT();
  const { control } = useFormContext<AccountingControl>();
  const [oandaOpen, setOandaOpen] = useState(false);
  const currencyMode = useWatch({ control, name: "currencyMode" });
  const multiCurrency = currencyMode === currencyModeSchema.enum.MultiCurrency;
  const canReadIntegrations = usePermissions(Resource.Integration).canRead;

  const runtimeConfig = useQuery({
    ...integration.runtimeConfig(OANDA_INTEGRATION_TYPE),
    enabled: multiCurrency,
    throwOnError: false,
  });
  const ready = runtimeConfig.data?.ready ?? false;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Currency")}
        description={t("The currency the books are kept and reported in.")}
      >
        <SelectField
          control={control}
          name="currencyMode"
          label={t("Currency mode")}
          description={t(
            "Single currency keeps every transaction in the functional currency; multi-currency takes transactions in others and converts them.",
          )}
          options={currencyModeChoices}
          required
        />
        <SelectField
          control={control}
          name="functionalCurrencyCode"
          label={t("Functional currency")}
          description={t("The currency the organization's accounts are kept and reported in.")}
          options={currencyChoices}
          required
        />
        <SettingsNested show={multiCurrency}>
          <OandaReadiness
            loading={runtimeConfig.isLoading}
            ready={ready}
            enabled={runtimeConfig.data?.enabled ?? false}
            configured={runtimeConfig.data?.configured ?? false}
            hasApiKey={runtimeConfig.data?.missingRequiredFields.includes("apiKey") === false}
            canReadIntegrations={canReadIntegrations}
            onConfigure={() => setOandaOpen(true)}
          />
        </SettingsNested>
        <SettingsNested show={multiCurrency && ready}>
          <SelectField
            control={control}
            name="exchangeRateDatePolicy"
            label={t("Exchange rate date")}
            description={t("Which date picks the exchange rate for a foreign-currency entry.")}
            options={exchangeRateDatePolicyChoices}
            required
          />
          <SelectField
            control={control}
            name="exchangeRateOverridePolicy"
            label={t("Exchange rate overrides")}
            description={t(
              "Whether people may replace the rate on an entry, and whether that needs approval.",
            )}
            options={exchangeRateOverridePolicyChoices}
            required
          />
          <GLAccountAutocompleteField
            control={control}
            name="realizedFxGainAccountId"
            label={t("Realized FX gain account")}
            placeholder={t("Select FX gain account")}
            description={t("Credited for exchange gains realized when a foreign amount settles.")}
            required={multiCurrency}
          />
          <GLAccountAutocompleteField
            control={control}
            name="realizedFxLossAccountId"
            label={t("Realized FX loss account")}
            placeholder={t("Select FX loss account")}
            description={t("Debited for exchange losses realized when a foreign amount settles.")}
            required={multiCurrency}
          />
        </SettingsNested>
      </SettingsSection>
      <OANDAExchangeRatesIntegrationModal
        open={oandaOpen && canReadIntegrations}
        onOpenChange={setOandaOpen}
      />
    </SettingsSections>
  );
}

/** Whether OANDA, which supplies the exchange rates, is connected: one row of the list. */
function OandaReadiness({
  loading,
  ready,
  enabled,
  configured,
  hasApiKey,
  canReadIntegrations,
  onConfigure,
}: {
  loading: boolean;
  ready: boolean;
  enabled: boolean;
  configured: boolean;
  hasApiKey: boolean;
  canReadIntegrations: boolean;
  onConfigure: () => void;
}) {
  const t = useT();

  let description = t("Connect OANDA to set exchange rate policies and FX accounts.");
  if (loading) {
    description = t("Checking the OANDA connection.");
  } else if (ready) {
    description = t("Exchange rates come from OANDA.");
  } else if (enabled && configured && hasApiKey) {
    description = t("Turn the OANDA integration on to set exchange rate policies and FX accounts.");
  }

  return (
    <SettingRow
      label={
        <span className="inline-flex items-center gap-2">
          <span
            aria-hidden
            className={cn(
              "size-1.5 shrink-0 rounded-full",
              ready ? "bg-success" : "bg-muted-foreground/50",
            )}
          />
          {ready ? t("OANDA connected") : t("OANDA required")}
        </span>
      }
      description={<p className={settingRowDescriptionClass}>{description}</p>}
      controlSize="auto"
    >
      <Button
        type="button"
        variant={ready ? "secondary" : "default"}
        size="sm"
        onClick={onConfigure}
        disabled={!canReadIntegrations}
        title={canReadIntegrations ? t("Configure OANDA") : t("Integration permission required")}
      >
        <Sliders04Icon />
        {t("Configure")}
      </Button>
    </SettingRow>
  );
}

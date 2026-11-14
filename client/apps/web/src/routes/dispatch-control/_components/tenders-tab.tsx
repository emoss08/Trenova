import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { tenderExhaustedActionChoices } from "@/lib/choices";
import type { DispatchControl } from "@/types/dispatch-control";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** How a tender answers a carrier that asks for another rate, and what it does once every carrier has passed. */
export function TendersTab() {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const marginsWithheld = !useProtectedFieldAccess("Margins");

  const counterOffersEnabled = useWatch({ control, name: "tenderCounterOffersEnabled" });
  const exhaustedAction = useWatch({ control, name: "tenderExhaustedAction" });
  const escalates = exhaustedAction === "Rebroadcast" || exhaustedAction === "Rewaterfall";

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Counter-offers")}
        description={t("How a tender handles a carrier asking for a different rate.")}
      >
        <FormControl>
          <SwitchField
            control={control}
            name="tenderCounterOffersEnabled"
            label={t("Take counter-offers")}
            description={t(
              "Carriers can answer an offer with a rate of their own instead of only accepting or declining.",
            )}
            position="left"
          />
        </FormControl>
        {counterOffersEnabled ? (
          <SettingsNested>
            <FormGroup cols={2}>
              <FormControl>
                <NumberField
                  control={control}
                  name="tenderCounterMarginFloorPercent"
                  valueType="string"
                  label={t("Margin floor")}
                  description={
                    marginsWithheld
                      ? t("Your role does not include margins.")
                      : t(
                          "A counter that leaves less margin than this on the shipment's charges needs someone who can approve tenders.",
                        )
                  }
                  placeholder={marginsWithheld ? t("Hidden") : "10"}
                  disabled={marginsWithheld}
                  sideText="%"
                  decimalScale={4}
                  allowNegative
                  min={-100}
                  max={100}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="tenderCounterDecisionMinutes"
                  label={t("Dispatch has to answer a counter within")}
                  description={t(
                    "A counter nobody answers in time lapses and the tender moves on to the next carrier.",
                  )}
                  placeholder="60"
                  sideText="minutes"
                  min={5}
                  max={1440}
                />
              </FormControl>
            </FormGroup>
            <FormControl>
              <SwitchField
                control={control}
                name="tenderAutoAcceptCounters"
                label={t("Take counters at or above the floor automatically")}
                description={t(
                  "The carrier is assigned at its rate without waiting for dispatch. Counters below the floor always wait for approval.",
                )}
                position="left"
              />
            </FormControl>
          </SettingsNested>
        ) : null}
      </SettingsSection>

      <SettingsSection
        title={t("Exhausted tenders")}
        description={t("What happens when every carrier on a tender has passed.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="tenderExhaustedAction"
              label={t("When every carrier passes")}
              description={t(
                "Dispatch is always told. An escalation offers the move again to the carriers on the tender, leaving out any that backed out.",
              )}
              options={tenderExhaustedActionChoices}
            />
          </FormControl>
        </FormGroup>
        {escalates ? (
          <SettingsNested>
            <FormGroup cols={2}>
              <FormControl>
                <NumberField
                  control={control}
                  name="tenderEscalationRateIncreasePercent"
                  valueType="string"
                  label={t("Raise the rates by up to")}
                  description={t(
                    "Each carrier's rate goes up by this much, but never past the margin floor. With no charges on the shipment, rates stay as they were.",
                  )}
                  placeholder="0"
                  sideText="%"
                  decimalScale={4}
                  min={0}
                  max={100}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="tenderEscalationOfferTtlMinutes"
                  label={t("Escalated offers stay open for")}
                  placeholder="60"
                  sideText="minutes"
                  min={5}
                  max={10080}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="tenderMaxEscalations"
                  label={t("Escalate at most")}
                  description={t("Rounds after the first tender; at most 5.")}
                  placeholder="1"
                  sideText="times"
                  min={0}
                  max={5}
                />
              </FormControl>
            </FormGroup>
          </SettingsNested>
        ) : null}
      </SettingsSection>
    </SettingsSections>
  );
}

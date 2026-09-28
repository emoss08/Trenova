import {
  pairingCodeFormSchema,
  type PairingCodeFormInput,
  type PairingCodeFormOutput,
} from "@/components/capture/capture-forms";
import { InputField } from "@/components/fields/input-field";
import { SectionPanel } from "@/components/section-panel";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { FormProvider, useForm } from "react-hook-form";

/**
 * Where the code from the companion goes. It takes the code however it was
 * typed or pasted, and says what a code looks like when it is not one,
 * rather than leaving the button greyed out with no reason.
 */
export function PairingCodeForm({
  initialCode,
  disabled,
  prominent,
  onLookUp,
}: {
  initialCode: string;
  disabled: boolean;
  /** The button is the page's next step until a computer is on screen. */
  prominent: boolean;
  onLookUp: (code: string) => void;
}) {
  const t = useT();
  const schema = useMemo(() => pairingCodeFormSchema(t), [t]);
  const form = useForm<PairingCodeFormInput, unknown, PairingCodeFormOutput>({
    resolver: zodResolver(schema),
    defaultValues: { code: initialCode },
  });

  return (
    <SectionPanel title={t("Code from Trenova Capture")}>
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void form.handleSubmit((values) => onLookUp(values.code))(event);
          }}
        >
          <div className="p-3">
            <FormGroup cols={1}>
              <FormControl>
                <InputField<PairingCodeFormInput>
                  control={form.control}
                  name="code"
                  label={t("Pairing code")}
                  aria-label={t("Pairing code")}
                  description={t("The eight letters Trenova Capture shows, like ABCD-EFGH")}
                  placeholder="ABCD-EFGH"
                  autoComplete="off"
                  autoCapitalize="characters"
                  spellCheck={false}
                  inputClassProps="font-mono uppercase"
                  disabled={disabled}
                />
              </FormControl>
            </FormGroup>
          </div>
          <div className="border-border flex justify-end gap-2 border-t px-3 py-2">
            <Button type="submit" variant={prominent ? "default" : "outline"} disabled={disabled}>
              {t("Look up")}
            </Button>
          </div>
        </Form>
      </FormProvider>
    </SectionPanel>
  );
}

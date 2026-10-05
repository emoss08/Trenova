import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import type { SelectOption } from "@/lib/graphql/select-options";
import type { OnboardingFormValues } from "@/types/onboarding";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, type KeyboardEvent } from "react";
import {
  Controller,
  useFormContext,
  useFormState,
  useWatch,
  type FieldPath,
} from "react-hook-form";
import { Kbd } from "./keyboard-hint";
import { useShake } from "./use-shake";

const ADDRESS_FIELDS = [
  "organization.addressLine1",
  "organization.city",
  "organization.stateId",
  "organization.postalCode",
] as const satisfies readonly FieldPath<OnboardingFormValues>[];

/** A message the server sent for a field, rather than one the form's own schema wrote. */
function serverMessage(error: { type?: unknown; message?: string } | undefined) {
  return error?.type === "validation" ? error.message : undefined;
}

function TextInput({
  name,
  placeholder,
  shaking,
  inputRef,
  onEnter,
  mono,
  digits,
  autoComplete,
}: {
  name: (typeof ADDRESS_FIELDS)[number];
  placeholder: string;
  shaking: boolean;
  inputRef?: React.Ref<HTMLInputElement>;
  onEnter: (event: KeyboardEvent<HTMLInputElement>) => void;
  mono?: boolean;
  digits?: number;
  autoComplete?: string;
}) {
  const { control } = useFormContext<OnboardingFormValues>();
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <input
          ref={(node) => {
            field.ref(node);
            if (typeof inputRef === "function") {
              inputRef(node);
            } else if (inputRef) {
              inputRef.current = node;
            }
          }}
          id={name}
          name={field.name}
          className={mono ? "nv-in nv-mono" : "nv-in"}
          value={field.value ?? ""}
          onChange={(event) =>
            field.onChange(
              digits ? event.target.value.replace(/\D/g, "").slice(0, digits) : event.target.value,
            )
          }
          onBlur={field.onBlur}
          onKeyDown={onEnter}
          placeholder={placeholder}
          aria-invalid={fieldState.invalid || undefined}
          data-shake={fieldState.invalid && shaking}
          inputMode={digits ? "numeric" : undefined}
          autoComplete={autoComplete}
        />
      )}
    />
  );
}

/** The headquarters turn: street, city, state and ZIP in one card. */
export function AddressAsk({
  onSubmit,
  onStateOptionChange,
}: {
  onSubmit: () => Promise<boolean>;
  onStateOptionChange: (option: SelectOption | null) => void;
}) {
  const t = useT();
  const { control, getFieldState } = useFormContext<OnboardingFormValues>();
  const formState = useFormState({ control, name: [...ADDRESS_FIELDS] });
  const [stateId, postalCode] = useWatch({
    control,
    name: ["organization.stateId", "organization.postalCode"],
  });
  const [shaking, shake] = useShake();
  const streetRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    streetRef.current?.focus({ preventScroll: true });
  }, []);

  const submit = async () => {
    if (!(await onSubmit())) {
      shake();
    }
  };

  const onEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void submit();
    }
  };

  const failing = ADDRESS_FIELDS.filter((name) => getFieldState(name, formState).invalid);
  const fromServer = failing
    .map((name) => serverMessage(getFieldState(name, formState).error))
    .find(Boolean);
  const errorText =
    failing.length === 0
      ? undefined
      : fromServer
        ? fromServer
        : failing.length === 1 && failing[0] === "organization.postalCode" && postalCode
          ? t("ZIP code needs 5 digits.")
          : t("Fill in the highlighted fields.");

  return (
    <div className="nv-fc">
      <div className="nv-fg">
        <div className="nv-fl nv-full">
          <label htmlFor="organization.addressLine1">{t("Street address")}</label>
          <TextInput
            name="organization.addressLine1"
            placeholder={t("1200 Industrial Pkwy")}
            shaking={shaking}
            inputRef={streetRef}
            onEnter={onEnter}
            autoComplete="address-line1"
          />
        </div>
        <div className="nv-fl">
          <label htmlFor="organization.city">{t("City")}</label>
          <TextInput
            name="organization.city"
            placeholder={t("Whitsett")}
            shaking={shaking}
            onEnter={onEnter}
            autoComplete="address-level2"
          />
        </div>
        <div
          className="nv-fl nv-state"
          data-empty={!stateId}
          data-invalid={getFieldState("organization.stateId", formState).invalid}
          data-shake={shaking}
        >
          <span className="nv-label">{t("State")}</span>
          <UsStateAutocompleteField
            control={control}
            name="organization.stateId"
            placeholder={t("Select")}
            triggerClassName="nv-in"
            onOptionChange={onStateOptionChange}
          />
        </div>
        <div className="nv-fl">
          <label htmlFor="organization.postalCode">{t("ZIP")}</label>
          <TextInput
            name="organization.postalCode"
            placeholder={t("27377")}
            shaking={shaking}
            onEnter={onEnter}
            mono
            digits={5}
            autoComplete="postal-code"
          />
        </div>
      </div>
      <div className="nv-fc-f">
        {errorText ? (
          <p className="nv-err" role="alert">
            {errorText}
          </p>
        ) : (
          <span className="nv-hint">
            <Kbd>{"↵"}</Kbd> {t("to continue")}
          </span>
        )}
        <span className="nv-sp" />
        <button type="button" className="nv-bt" data-ink="true" onClick={() => void submit()}>
          {t("Continue")}
          <ArrowRightIcon size={14} strokeWidth={1.7} aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}

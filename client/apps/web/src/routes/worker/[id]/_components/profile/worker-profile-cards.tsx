import {
  FleetCodeAutocompleteField,
  JobPositionAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { CustomFieldsSection } from "@/components/custom-fields-section";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { FieldLayoutProvider, FieldNote } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  cdlClassChoices,
  driverTypeChoices,
  endorsementTypeChoices,
  genderChoices,
  workerTypeChoices,
} from "@/lib/choices";
import { InfoCircleIcon, Lock01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useNow } from "@trenova/shared/hooks/use-now";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { daysUntil, formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { ComplianceStatus, Worker } from "@trenova/shared/types/worker";
import { ComplianceBadge } from "../compliance-badge";
import type { ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { LicenceLine } from "./licence-line";
import { WorkerHomeAddressField } from "./worker-home-address-field";

/** A titled card of the Profile form, the factoring details card's anatomy. */
function ProfileCard({
  title,
  description,
  trailing,
  children,
}: {
  title: string;
  description: string;
  /** At the header's right, wrapping under the title when narrow (the licence line). */
  trailing?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section aria-label={title} className="min-w-0 rounded-lg border bg-card">
      <header className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b px-4 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="text-sm font-semibold">{title}</h3>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
        {trailing ? <div className="ml-auto max-w-full min-w-0">{trailing}</div> : null}
      </header>
      <div className="flex flex-col gap-3 p-4">{children}</div>
    </section>
  );
}

/** A value the form shows but another part of the record sets: dashed, with a lock. */
export function LockedRow({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex items-center gap-3 rounded-md border border-dashed px-3 py-2.5">
      <Lock01Icon aria-hidden className="size-3.5 shrink-0 text-muted-foreground" />
      <p className="min-w-0 flex-1 text-xs text-muted-foreground">{children}</p>
      {action}
    </div>
  );
}

function ColumnHeading({ children }: { children: ReactNode }) {
  return <h2 className="-mb-2 text-xs font-medium text-foreground-subtle">{children}</h2>;
}

/**
 * The medical card's expiry, with how soon it runs out under it as the field's own description:
 * danger once lapsed, warning inside 90 days.
 */
function MedicalCardExpiryField() {
  const t = useT();
  const { control } = useFormContext<Worker>();
  const expiry = useWatch({ control, name: "profile.medicalCardExpiry" });
  const now = useNow("minute");
  const days = expiry ? daysUntil(expiry, { now }) : null;
  return (
    <AutoCompleteDateField
      control={control}
      name="profile.medicalCardExpiry"
      label={t("Medical card expires")}
      placeholder={t("MM/DD/YYYY")}
      description={
        days === null ? undefined : (
          <FieldNote live tone={days < 0 ? "danger" : days <= 90 ? "warning" : "muted"}>
            {days < 0
              ? t("{0, plural, one {Expired # day ago} other {Expired # days ago}}", -days)
              : days === 0
                ? t("Expires today")
                : t("{0, plural, one {Expires in # day} other {Expires in # days}}", days)}
          </FieldNote>
        )
      }
    />
  );
}

export type WorkerProfileCardsProps = {
  /** `edit` draws the locked status, termination and compliance rows; `create` leaves them out. */
  mode: "edit" | "create";
  /** The reader's role withholds personal details: they show as hidden and stop being required. */
  personalWithheld: boolean;
  /** What the record holds now, for the locked rows and the licence line's stored state. */
  stored?: {
    status: string;
    terminationDate: number | null;
    complianceStatus: ComplianceStatus | null;
    licenseStateId: string | null;
    licenseStateAbbreviation: string | null;
  };
  /** Opens the employment event form; the status row offers it when given. */
  onRecordEvent?: () => void;
  /** A note at the top of the right column (New worker says what creation starts). */
  note?: ReactNode;
};

/**
 * The worker's own record as two columns of cards: the person (identity, home, contact, custom
 * fields) and their licence and clearance (dates, CDL, medical card, qualification, TWIC). The
 * Profile tab and New worker draw the same cards from one form.
 */
export function WorkerProfileCards({
  mode,
  personalWithheld,
  stored,
  onRecordEvent,
  note,
}: WorkerProfileCardsProps) {
  const t = useT();
  const rt = useRichT();
  const { control } = useFormContext<Worker>();
  const compliance = usePlanFeature("workforce.compliance").available;
  const [endorsement, qualified] = useWatch({
    control,
    name: ["profile.endorsement", "profile.isQualified"],
  });
  const requiresHazmatExpiry = endorsement === "H" || endorsement === "X";
  const withheld = t("Your role does not include personal details.");
  const strong = (chunks: ReactNode) => <b className="font-medium text-foreground">{chunks}</b>;

  return (
    <div className="grid items-start gap-4 xl:grid-cols-2">
      <div className="flex min-w-0 flex-col gap-4">
        <ColumnHeading>{t("The person")}</ColumnHeading>
        <ProfileCard
          title={t("Identity")}
          description={t("Who they are and how they are employed.")}
        >
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                required
                name="firstName"
                label={t("First name")}
                placeholder={t("First name")}
                description={t("Legal first name as it appears on the CDL.")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                required
                name="lastName"
                label={t("Last name")}
                placeholder={t("Last name")}
                description={t("Legal last name as it appears on the CDL.")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                options={genderChoices}
                required
                name="gender"
                label={t("Gender")}
                placeholder={t("Select")}
                description={t("Gender as recorded on the license.")}
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.dob"
                label={t("Date of birth")}
                required={!personalWithheld}
                description={
                  personalWithheld ? withheld : t("Date of birth as shown on the license.")
                }
                placeholder={personalWithheld ? t("Hidden") : t("MM/DD/YYYY")}
                disabled={personalWithheld}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                options={workerTypeChoices}
                required
                name="type"
                label={t("Worker type")}
                placeholder={t("Select")}
                description={t("Whether the worker is an employee or contractor.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                options={driverTypeChoices}
                required
                name="driverType"
                label={t("Driver type")}
                placeholder={t("Select")}
                description={t("Local, regional, over the road or team.")}
              />
            </FormControl>
            <FormControl>
              <FleetCodeAutocompleteField<Worker>
                name="fleetCodeId"
                control={control}
                clearable
                label={t("Fleet code")}
                placeholder={t("Search fleet codes")}
                description={t("The fleet code the worker runs under.")}
              />
            </FormControl>
            <FormControl>
              <JobPositionAutocompleteField<Worker>
                name="positionId"
                control={control}
                driving
                clearable
                label={t("Position")}
                placeholder={t("Search positions")}
                description={t(
                  "The title the roster is counted by. Front-office titles are held by users, not workers.",
                )}
              />
            </FormControl>
          </FormGroup>
          {mode === "edit" && stored ? (
            <LockedRow
              action={
                onRecordEvent ? (
                  <Button type="button" size="sm" variant="secondary" onClick={onRecordEvent}>
                    {t("Record event")}
                  </Button>
                ) : null
              }
            >
              {rt(
                "<b>Status: {0}.</b> Employment status moves through employment events.",
                { b: strong },
                stored.status === "Active" ? t("Active") : t("Inactive"),
              )}
            </LockedRow>
          ) : null}
        </ProfileCard>

        <ProfileCard
          title={t("Home address")}
          description={t("Where they live and when they were last home.")}
        >
          <FormGroup cols={2}>
            <FormControl cols="full">
              <WorkerHomeAddressField />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="addressLine2"
                label={t("Apartment or suite")}
                placeholder={t("Apartment, suite, unit")}
                maxLength={150}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                required
                name="city"
                label={t("City")}
                placeholder={t("City")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <UsStateAutocompleteField
                control={control}
                name="stateId"
                label={t("State")}
                required
                placeholder={t("Search states")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                required
                name="postalCode"
                label={t("Postal code")}
                placeholder={t("e.g. 60432")}
                description={t("5-digit ZIP code (or ZIP+4).")}
                maxLength={10}
                inputMode="numeric"
              />
            </FormControl>
            <FormControl cols="full">
              <AutoCompleteDateField
                control={control}
                name="lastHomeAt"
                label={t("Last home")}
                description={t("Dispatch plans home time from this.")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
          </FormGroup>
        </ProfileCard>

        <ProfileCard title={t("Contact")} description={t("How to reach them and who to call.")}>
          <FormGroup cols={2}>
            <FormControl cols="full">
              <InputField
                control={control}
                name="email"
                label={t("Email")}
                placeholder={t("name@example.com")}
                maxLength={255}
              />
            </FormControl>
            <FormControl>
              <PhoneNumberField
                control={control}
                name="phoneNumber"
                required
                label={t("Phone")}
                placeholder="(555) 555-0100"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="emergencyContactName"
                label={t("Emergency contact")}
                placeholder={t("Full name")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <PhoneNumberField
                control={control}
                name="emergencyContactPhone"
                label={t("Emergency contact phone")}
                placeholder="(555) 555-0100"
              />
            </FormControl>
          </FormGroup>
        </ProfileCard>

        <div className="min-w-0 rounded-lg border bg-card p-4 empty:hidden">
          <CustomFieldsSection resourceType="worker" control={control} />
        </div>
      </div>

      <div className="flex min-w-0 flex-col gap-4">
        <ColumnHeading>{t("License and clearance")}</ColumnHeading>
        {note}
        <ProfileCard
          title={t("Employment dates")}
          description={t("When they started, and when they left.")}
        >
          <FormGroup cols={2}>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.hireDate"
                label={t("Hire date")}
                required
                description={t("Tenure is counted from here.")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
          </FormGroup>
          {mode === "edit" && stored ? (
            <LockedRow>
              {stored.terminationDate
                ? rt(
                    "<b>Termination date: {0}.</b> Set by a Terminated event.",
                    { b: strong },
                    formatUnixDateMedium(stored.terminationDate),
                  )
                : rt("<b>Termination date: not terminated.</b> Set by a Terminated event.", {
                    b: strong,
                  })}
            </LockedRow>
          ) : null}
        </ProfileCard>

        <ProfileCard
          title={t("License")}
          description={t("Their CDL, written the way the card reads.")}
          trailing={
            <LicenceLine
              control={control}
              storedStateId={stored?.licenseStateId ?? null}
              storedStateAbbreviation={stored?.licenseStateAbbreviation ?? null}
            />
          }
        >
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                required={!personalWithheld}
                name="profile.licenseNumber"
                label={t("License number")}
                placeholder={personalWithheld ? t("Hidden") : t("As printed on the card")}
                description={personalWithheld ? withheld : undefined}
                maxLength={50}
                disabled={personalWithheld}
                inputClassProps="font-mono"
              />
            </FormControl>
            <FormControl>
              <UsStateAutocompleteField
                control={control}
                name="profile.licenseStateId"
                label={t("Issuing state")}
                placeholder={t("Search states")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                options={cdlClassChoices}
                required
                name="profile.cdlClass"
                label={t("CDL class")}
                placeholder={t("Select")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                options={endorsementTypeChoices}
                required
                name="profile.endorsement"
                label={t("Endorsement")}
                placeholder={t("Select")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="profile.cdlRestrictions"
                label={t("Restrictions")}
                placeholder={t("e.g. L, Z")}
                maxLength={100}
                inputClassProps="font-mono"
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.licenseExpiry"
                label={t("License expires")}
                required
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
            {requiresHazmatExpiry ? (
              <FormControl className="animate-expand-in motion-reduce:animate-none">
                <AutoCompleteDateField
                  control={control}
                  name="profile.hazmatExpiry"
                  label={t("Hazmat endorsement expires")}
                  required
                  description={t("Shown for a hazmat endorsement, H or X.")}
                  placeholder={t("MM/DD/YYYY")}
                />
              </FormControl>
            ) : null}
          </FormGroup>
        </ProfileCard>

        <ProfileCard
          title={t("Medical certificate")}
          description={t("Their DOT physical and who signed it.")}
        >
          <FormGroup cols={2}>
            <FormControl>
              <MedicalCardExpiryField />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.physicalDueDate"
                label={t("Next physical due")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="profile.medicalExaminerName"
                label={t("Medical examiner")}
                placeholder={t("Name on the certificate")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="profile.medicalExaminerNpi"
                label={t("Examiner NPI")}
                placeholder={t("10 digits")}
                maxLength={20}
                inputMode="numeric"
                inputClassProps="font-mono"
              />
            </FormControl>
          </FormGroup>
        </ProfileCard>

        <ProfileCard
          title={t("Qualification")}
          description={t("Whether dispatch may put them on a load.")}
        >
          {compliance ? (
            <LockedRow>
              <span className="inline-flex flex-wrap items-center gap-2">
                <span className="font-medium text-foreground">{t("Compliance")}</span>
                <ComplianceBadge status={stored?.complianceStatus ?? "Pending"} />
                <span>
                  {mode === "edit"
                    ? t("Graded from the worker's credentials.")
                    : t("Graded from the worker's credentials once they are on file.")}
                </span>
              </span>
            </LockedRow>
          ) : null}
          <FormGroup cols={2}>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.mvrDueDate"
                label={t("MVR review due")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
          </FormGroup>
          <FieldLayoutProvider layout="row">
            <div className="divide-y rounded-md border">
              <SwitchField
                control={control}
                name="profile.isQualified"
                label={t("Qualified to drive")}
                description={t("They can be put on loads. Off disqualifies them.")}
              />
              <SwitchField
                control={control}
                name="availableForDispatch"
                label={t("Available for dispatch")}
                description={t("Dispatch sees them as free to plan.")}
              />
              <SwitchField
                control={control}
                name="canBeAssigned"
                label={t("Can be assigned to equipment")}
                description={t("They can be given a tractor or trailer.")}
              />
              <SwitchField
                control={control}
                name="profile.eldExempt"
                label={t("ELD exempt")}
                description={t("They log on paper instead of an ELD.")}
              />
              <SwitchField
                control={control}
                name="profile.shortHaulExempt"
                label={t("Short haul exempt")}
                description={t("Within 150 air miles they keep no log.")}
              />
            </div>
          </FieldLayoutProvider>
          {qualified === false ? (
            <div className="animate-expand-in motion-reduce:animate-none">
              <InputField
                control={control}
                name="profile.disqualificationReason"
                label={t("Disqualification reason")}
                placeholder={t("Why they cannot drive")}
                maxLength={255}
              />
            </div>
          ) : null}
        </ProfileCard>

        <ProfileCard title={t("TWIC")} description={t("Port access card.")}>
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="profile.twicCardNumber"
                label={t("Card number")}
                placeholder={personalWithheld ? t("Hidden") : t("0000-0000-0000")}
                description={personalWithheld ? withheld : undefined}
                maxLength={50}
                disabled={personalWithheld}
                inputClassProps="font-mono"
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name="profile.twicExpiry"
                label={t("Expires")}
                placeholder={t("MM/DD/YYYY")}
              />
            </FormControl>
          </FormGroup>
        </ProfileCard>
      </div>
    </div>
  );
}

/** The note New worker puts above the licence column: what creating the worker starts. */
export function NewWorkerNote() {
  const rt = useRichT();
  return (
    <div className="flex gap-2.5 rounded-md border bg-card px-3.5 py-3 text-xs text-muted-foreground">
      <InfoCircleIcon aria-hidden className="mt-0.5 size-3.5 shrink-0 text-foreground-subtle" />
      <p>
        {rt(
          "<b>After you create them:</b> the onboarding checklist starts, required training is assigned and the default time off policy applies.",
          { b: (chunks: ReactNode) => <b className="font-medium text-foreground">{chunks}</b> },
        )}
      </p>
    </div>
  );
}

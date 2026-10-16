import { useT } from "@trenova/shared/i18n/use-t";
import { EmailProfileAutocompleteField } from "@/components/autocomplete-fields";
import { FormSaveDock } from "@/components/form-save-dock";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { email } from "@/lib/queries/email";
import { apiService } from "@/services/api";
import type { EmailProfileAssignment } from "@trenova/shared/types/email";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { z } from "zod";
import { emailPurposes } from "./email-profile-constants";
import { translate } from "@trenova/shared/i18n/runtime";

// A cleared purpose holds null, which is how the autocomplete clears; it saves as no
// assignment.
const assignedProfileSchema = z.string().nullable();

export const purposeAssignmentsSchema = z.object({
  General: assignedProfileSchema,
  Billing: assignedProfileSchema,
  Reporting: assignedProfileSchema,
  Operations: assignedProfileSchema,
  Authentication: assignedProfileSchema,
  Notifications: assignedProfileSchema,
});

export type PurposeAssignmentsFormValues = z.infer<typeof purposeAssignmentsSchema>;

const emptyAssignments = Object.fromEntries(
  emailPurposes.map((purpose) => [purpose, ""]),
) as PurposeAssignmentsFormValues;

function toFormValues(assignments: EmailProfileAssignment[]): PurposeAssignmentsFormValues {
  const values = { ...emptyAssignments };
  for (const assignment of assignments) {
    if (emailPurposes.includes(assignment.purpose) && assignment.profileId) {
      values[assignment.purpose] = assignment.profileId;
    }
  }
  return values;
}

export function toPayload(values: PurposeAssignmentsFormValues): EmailProfileAssignment[] {
  const assignments: EmailProfileAssignment[] = [];
  for (const purpose of emailPurposes) {
    const profileId = values[purpose];
    if (profileId) {
      assignments.push({ purpose, profileId });
    }
  }
  return assignments;
}

export function PurposeAssignmentsPanel() {
  const t = useT();

  const queryClient = useQueryClient();
  const assignmentsQuery = useQuery(email.assignments());
  const form = useForm<PurposeAssignmentsFormValues>({
    resolver: zodResolver(purposeAssignmentsSchema),
    defaultValues: emptyAssignments,
  });

  useEffect(() => {
    form.reset(toFormValues(assignmentsQuery.data ?? []));
  }, [assignmentsQuery.data, form]);

  const saveAssignments = useMutation({
    mutationFn: (values: PurposeAssignmentsFormValues) =>
      apiService.emailService.updateAssignments(toPayload(values)),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: email.assignments().queryKey });
    },
    onError: (error) => {
      toast.error(t("Failed to update purpose assignments"), {
        description:
          error instanceof Error ? error.message : translate("An unexpected error occurred"),
      });
    },
  });

  return (
    <section className="border-border bg-background rounded-md border">
      <div className="border-border flex flex-col gap-1 border-b px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-medium">
          {t("Purpose assignments")}
        </div>
        <p className="text-muted-foreground text-xs">
          {t(
            "Assign each email purpose to an active sender profile. Clearing a purpose removes its assignment on save.",
          )}
        </p>
      </div>
      <div className="p-4">
        <FormProvider {...form}>
          <Form onSubmit={form.handleSubmit((values) => saveAssignments.mutate(values))}>
            <FormGroup cols={3}>
              {emailPurposes.map((purpose) => (
                <FormControl key={purpose}>
                  <EmailProfileAutocompleteField<PurposeAssignmentsFormValues>
                    control={form.control}
                    name={purpose}
                    label={purpose}
                    placeholder={t("Unassigned")}
                    clearable
                    noResultsMessage={t("No active email profiles found.")}
                    initialLimit={20}
                  />
                </FormControl>
              ))}
            </FormGroup>
            <FormSaveDock saveButtonContent={t("Save assignments")} showHeightGap={false} />
          </Form>
        </FormProvider>
      </div>
    </section>
  );
}

import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  createTrainingCourse,
  TRAINING_COURSE_LIST_KEY,
  updateTrainingCourse,
  type TrainingCourseRow,
} from "@/lib/graphql/worker-training";
import type { TrainingCourseInput } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  trainingCourseFormSchema,
  type TrainingCourseFormValues,
} from "@trenova/shared/types/worker-training";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type Resolver } from "react-hook-form";
import { TrainingCourseForm } from "./training-course-form";

export function buildTrainingCourseDefaults(
  row?: TrainingCourseRow | null,
): TrainingCourseFormValues {
  if (!row) {
    return {
      code: "",
      name: "",
      description: null,
      category: "Safety",
      status: "Active",
      delivery: "Online",
      contentUrl: null,
      durationMinutes: 60,
      passingScore: null,
      validityMonths: null,
      renewalWindowDays: 30,
      isRequired: false,
      requiredForDriverTypes: [],
      dueDaysAfterAssignment: 30,
      requiresAcknowledgement: true,
    };
  }
  return {
    code: row.code,
    name: row.name,
    description: row.description ?? null,
    category: row.category,
    status: row.status === "Inactive" ? "Inactive" : "Active",
    delivery: row.delivery,
    contentUrl: row.contentUrl ?? null,
    durationMinutes: row.durationMinutes,
    passingScore: row.passingScore ?? null,
    validityMonths: row.validityMonths ?? null,
    renewalWindowDays: row.renewalWindowDays,
    isRequired: row.isRequired,
    requiredForDriverTypes: [...row.requiredForDriverTypes],
    dueDaysAfterAssignment: row.dueDaysAfterAssignment,
    requiresAcknowledgement: row.requiresAcknowledgement,
  };
}

export function toTrainingCourseInput(
  values: TrainingCourseFormValues,
  version?: number,
): TrainingCourseInput {
  return {
    code: values.code.toUpperCase(),
    name: values.name,
    description: values.description ?? undefined,
    category: values.category,
    status: values.status,
    delivery: values.delivery,
    contentUrl: values.contentUrl ?? undefined,
    durationMinutes: values.durationMinutes,
    passingScore: values.passingScore ?? undefined,
    validityMonths: values.validityMonths ?? undefined,
    renewalWindowDays: values.renewalWindowDays,
    isRequired: values.isRequired,
    requiredForDriverTypes: values.isRequired ? values.requiredForDriverTypes : [],
    dueDaysAfterAssignment: values.dueDaysAfterAssignment,
    requiresAcknowledgement: values.requiresAcknowledgement,
    version,
  };
}

export function TrainingCoursePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<TrainingCourseRow>) {
  if (mode === "edit" && row) {
    return <TrainingCourseEditPanel open={open} onOpenChange={onOpenChange} row={row} />;
  }
  return <TrainingCourseCreatePanel open={open} onOpenChange={onOpenChange} />;
}

function TrainingCourseCreatePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const form = useForm<TrainingCourseFormValues>({
    resolver: zodResolver(trainingCourseFormSchema) as Resolver<TrainingCourseFormValues>,
    defaultValues: buildTrainingCourseDefaults(null),
  });

  return (
    <FormCreatePanel<TrainingCourseFormValues, TrainingCourseRow>
      open={open}
      onOpenChange={onOpenChange}
      title="Training Course"
      description="Add a course workers can be assigned, and decide whether it is required, how it is taken, and what passes."
      queryKey={TRAINING_COURSE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<TrainingCourseForm isEdit={false} />}
      mutationFn={async (values) => {
        await createTrainingCourse(toTrainingCourseInput(values));
        return values;
      }}
    />
  );
}

function TrainingCourseEditPanel({
  open,
  onOpenChange,
  row,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: TrainingCourseRow;
}) {
  const formRow = {
    ...row,
    ...buildTrainingCourseDefaults(row),
  } as unknown as TrainingCourseRow & Record<string, unknown>;
  const form = useForm<TrainingCourseFormValues>({
    resolver: zodResolver(trainingCourseFormSchema) as Resolver<TrainingCourseFormValues>,
    defaultValues: buildTrainingCourseDefaults(row),
  });

  return (
    <FormEditPanel<TrainingCourseFormValues, TrainingCourseRow & Record<string, unknown>>
      open={open}
      onOpenChange={onOpenChange}
      row={formRow}
      title="Training Course"
      fieldKey="code"
      queryKey={TRAINING_COURSE_LIST_KEY}
      form={form}
      size="lg"
      formComponent={<TrainingCourseForm isEdit openRecordCount={row.openRecordCount} />}
      mutationFn={async (values) => {
        await updateTrainingCourse(row.id, toTrainingCourseInput(values, row.version));
        return values;
      }}
    />
  );
}

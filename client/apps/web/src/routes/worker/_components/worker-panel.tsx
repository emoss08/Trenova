import type { DataTablePanelProps } from "@/types/data-table";
import { workerSchema, type Worker } from "@/types/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type UseFormReturn } from "react-hook-form";
import { WorkerCreatePanel } from "./worker-create-panel";
import { WorkerEditPanel } from "./worker-edit-panel";

export function WorkerPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<Worker>) {
  const form = useForm({
    resolver: zodResolver(workerSchema),
    defaultValues: {
      status: "Active",
      type: "Employee",
      driverType: "OTR",
      gender: "Male",
      firstName: "",
      lastName: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      postalCode: "",
      stateId: "",
      fleetCodeId: "",
      managerId: "",
      email: "",
      phoneNumber: "",
      emergencyContactName: "",
      emergencyContactPhone: "",
      externalId: "",
      profilePicUrl: "",
      canBeAssigned: false,
      availableForDispatch: true,
      assignmentBlocked: "",
      profile: {
        dob: undefined,
        hireDate: undefined,
        licenseNumber: "",
        licenseStateId: "",
        licenseExpiry: undefined,
        cdlClass: "A",
        cdlRestrictions: "",
        endorsement: "O",
        hazmatExpiry: undefined,
        medicalCardExpiry: undefined,
        medicalExaminerName: "",
        medicalExaminerNpi: "",
        twicCardNumber: "",
        twicExpiry: undefined,
        terminationDate: undefined,
        physicalDueDate: undefined,
        mvrDueDate: undefined,
        complianceStatus: "Pending",
        isQualified: false,
        disqualificationReason: "",
        lastComplianceCheck: 0,
        lastMvrCheck: 0,
        lastDrugTest: 0,
        eldExempt: false,
        shortHaulExempt: false,
      },
      id: undefined,
      version: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      state: undefined,
      fleetCode: undefined,
      pto: undefined,
    },
  });

  if (mode === "edit") {
    return (
      <WorkerEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form as UseFormReturn<Worker>}
      />
    );
  }

  return (
    <WorkerCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form as UseFormReturn<Worker>}
    />
  );
}

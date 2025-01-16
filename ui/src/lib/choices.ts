import { type ChoiceProps, Gender, Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import { Endorsement, PTOStatus, PTOType, WorkerType } from "@/types/worker";

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const statusChoices = [
  { value: Status.Active, label: "Active", color: "#15803d" },
  { value: Status.Inactive, label: "Inactive", color: "#b91c1c" },
] satisfies ReadonlyArray<ChoiceProps<Status>>;

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const workerTypeChoices = [
  { value: WorkerType.Employee, label: "Employee", color: "#15803d" },
  { value: WorkerType.Contractor, label: "Contractor", color: "#7e22ce" },
] satisfies ReadonlyArray<ChoiceProps<WorkerType>>;

export const endorsementChoices = [
  { value: Endorsement.None, label: "None", color: "#15803d" },
  { value: Endorsement.Tanker, label: "Tanker", color: "#7e22ce" },
  { value: Endorsement.Hazmat, label: "Hazmat", color: "#dc2626" },
  { value: Endorsement.TankerHazmat, label: "Tanker/Hazmat", color: "#f59e0b" },
  { value: Endorsement.Passenger, label: "Passenger", color: "#1d4ed8" },
  {
    value: Endorsement.DoublesTriples,
    label: "Doubles/Triples",
    color: "#0369a1",
  },
] satisfies ReadonlyArray<ChoiceProps<Endorsement>>;

export const equipmentClassChoices = [
  { value: EquipmentClass.Tractor, label: "Tractor", color: "#15803d" },
  { value: EquipmentClass.Trailer, label: "Trailer", color: "#7e22ce" },
  { value: EquipmentClass.Container, label: "Container", color: "#dc2626" },
  { value: EquipmentClass.Other, label: "Other", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<EquipmentClass>>;

export const genderChoices = [
  { value: Gender.Male, label: "Male", color: "#1d4ed8" },
  { value: Gender.Female, label: "Female", color: "#ec4899" },
] satisfies ReadonlyArray<ChoiceProps<Gender>>;

export const ptoStatusChoices = [
  { value: PTOStatus.Requested, label: "Requested", color: "#15803d" },
  { value: PTOStatus.Approved, label: "Approved", color: "#7e22ce" },
  { value: PTOStatus.Rejected, label: "Rejected", color: "#b91c1c" },
  { value: PTOStatus.Cancelled, label: "Cancelled", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<PTOStatus>>;

export const ptoTypeChoices = [
  { value: PTOType.Vacation, label: "Vacation", color: "#15803d" },
  { value: PTOType.Sick, label: "Sick", color: "#7e22ce" },
  { value: PTOType.Holiday, label: "Holiday", color: "#b91c1c" },
  { value: PTOType.Bereavement, label: "Bereavement", color: "#f59e0b" },
  { value: PTOType.Maternity, label: "Maternity", color: "#0369a1" },
  { value: PTOType.Paternity, label: "Paternity", color: "#0369a1" },
] satisfies ReadonlyArray<ChoiceProps<PTOType>>;

import { type ChoiceProps, Gender, Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
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

export const hazardousClassChoices = [
  {
    value: HazardousClassChoiceProps.HazardClass1And1,
    label: "Division 1.1: Mass Explosive Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And2,
    label: "Division 1.2: Projection Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And3,
    label: "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And4,
    label: "Division 1.4: Minor Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And5,
    label: "Division 1.5: Very Insensitive With Mass Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And6,
    label: "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And1,
    label: "Division 2.1: Flammable Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And2,
    label: "Division 2.2: Non-Flammable Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And3,
    label: "Division 2.3: Poisonous Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass3,
    label: "Division 3: Flammable Liquids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And1,
    label: "Division 4.1: Flammable Solids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And2,
    label: "Division 4.2: Spontaneously Combustible Solids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And3,
    label: "Division 4.3: Dangerous When Wet",
  },
  {
    value: HazardousClassChoiceProps.HazardClass5And1,
    label: "Division 5.1: Oxidizing Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass5And2,
    label: "Division 5.2: Organic Peroxides",
  },
  {
    value: HazardousClassChoiceProps.HazardClass6And1,
    label: "Division 6.1: Toxic Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass6And2,
    label: "Division 6.2: Infectious Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass7,
    label: "Division 7: Radioactive Material",
  },
  {
    value: HazardousClassChoiceProps.HazardClass8,
    label: "Division 8: Corrosive Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass9,
    label: "Division 9: Miscellaneous Hazardous Substances and Articles",
  },
] satisfies ReadonlyArray<ChoiceProps<HazardousClassChoiceProps>>;

export const packingGroupChoices = [
  {
    value: PackingGroupChoiceProps.PackingGroupI,
    label: "I (High Danger)",
    color: "#b91c1c",
  },
  {
    value: PackingGroupChoiceProps.PackingGroupII,
    label: "II (Medium Danger)",
    color: "#ca8a04",
  },
  {
    value: PackingGroupChoiceProps.PackingGroupIII,
    label: "III (Low Danger)",
    color: "#16a34a",
  },
] satisfies ReadonlyArray<ChoiceProps<PackingGroupChoiceProps>>;

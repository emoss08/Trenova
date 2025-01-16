import { Gender, Status } from "@/types/common";
import {
  ComplianceStatus,
  Endorsement,
  PTOStatus,
  PTOType,
  WorkerType,
} from "@/types/worker";
import { array, boolean, InferType, mixed, number, object, string } from "yup";

/* Worker Profile Schema */
const workerProfileSchema = object({
  dob: number().required("Date of birth is required"),
  licenseNumber: string().required("License number is required"),
  endorsement: mixed<Endorsement>()
    .required("Endorsement is required")
    .oneOf(Object.values(Endorsement)),
  hazmatExpiry: number().when("endorsement", {
    is: (value: Endorsement) =>
      value === Endorsement.Hazmat || value === Endorsement.TankerHazmat,
    then: (schema) => schema.required("Hazmat expiry is required"),
    otherwise: (schema) => schema.optional(),
  }),
  complianceStatus: mixed<ComplianceStatus>()
    .required("Compliance status is required")
    .oneOf(Object.values(ComplianceStatus)),
  isQualified: boolean().required("Is qualified is required"),
  licenseExpiry: number().required("License expiry is required"),
  hireDate: number().required("Hire date is required"),
  licenseStateId: string().required("License state is required"),
  terminationDate: number().optional().nullable(),
  physicalDueDate: number().optional().nullable(),
  mvrDueDate: number().optional().nullable(),
  lastMvrCheck: number().required("Last MVR check is required"),
  lastDrugTest: number().required("Last drug test is required"),
});

/* Worker PTO Schema */
const workerPTOSchema = object({
  status: mixed<PTOStatus>()
    .required("Status is required")
    .oneOf(Object.values(PTOStatus)),
  type: mixed<PTOType>()
    .required("Type is required")
    .oneOf(Object.values(PTOType)),
  startDate: number().min(1, "Start date is required"),
  endDate: number().min(1, "End date is required"),
  reason: string().optional(),
});

/* Worker Schema */
export const workerSchema = object({
  // Id is optional because it is not required when creating a new worker
  id: string().optional(),
  organizationId: string().optional(),
  businessUnitId: string().optional(),
  profilePictureUrl: string().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  type: mixed<WorkerType>()
    .required("Type is required")
    .oneOf(Object.values(WorkerType)),
  firstName: string().required("First name is required"),
  lastName: string().required("Last name is required"),
  addressLine1: string().required("Address line 1 is required"),
  addressLine2: string().optional(),
  city: string().required("City is required"),
  stateId: string().required("State is required"),
  fleetCodeId: string().optional(),
  gender: mixed<Gender>()
    .required("Gender is required")
    .oneOf(Object.values(Gender)),
  postalCode: string().required("Postal code is required"),
  profile: workerProfileSchema,
  pto: array().of(workerPTOSchema),
  createdAt: number().optional(),
  updatedAt: number().optional(),
});

export type WorkerSchema = InferType<typeof workerSchema>;

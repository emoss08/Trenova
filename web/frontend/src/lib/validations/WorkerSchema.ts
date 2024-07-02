import { type StatusChoiceProps } from "@/types";
import {
  EmploymentHistoryFormValues,
  EnumEmploymentVerificationStatus,
  EnumWorkerType,
  WorkerFormValues,
  type WorkerProfileFormValues,
} from "@/types/worker";
import { array, mixed, object, string, type ObjectSchema } from "yup";

const workerProfileSchema: ObjectSchema<WorkerProfileFormValues> =
  object().shape({
    race: string(),
    sex: string(),
    dateOfBirth: string().required("Date of Birth is required"),
    stateId: string().required("License State is required"),
    licenseExpirationDate: string().required(
      "License Expiration Date is required",
    ),
    endorsements: string().required("Endorsements is required"),
    hazmatExpirationDate: string().nullable(),
    hireDate: string().nullable(),
    terminationDate: string().nullable(),
    physicalDueDate: string().nullable(),
    mvrDueDate: string().nullable(),
  });

const employmentHistorySchema: ObjectSchema<EmploymentHistoryFormValues> =
  object().shape({
    employerName: string().required("Employer Name is required"),
    employerAddress: string().required("Employer Address is required"),
    employerContactInfo: string().required("Employer Contact Info is required"),
    startDate: string().required("Start Date is required"),
    endDate: string().required("End Date is required"),
    reasonForLeaving: string(),
    verificationStatus: mixed<EnumEmploymentVerificationStatus>()
      .required("Verification Status is required")
      .oneOf(Object.values(EnumEmploymentVerificationStatus)),
  });

export const workerSchema: ObjectSchema<WorkerFormValues> = object().shape({
  status: string<StatusChoiceProps>().required("Status is required"),
  code: string().required("Code is required"),
  workerType: mixed<EnumWorkerType>()
    .required("Worker Type is required")
    .oneOf(Object.values(EnumWorkerType)),
  firstName: string().required("First Name is required"),
  lastName: string().required("Last Name is required"),
  addressLine1: string(),
  addressLine2: string(),
  city: string(),
  stateId: string().nullable(),
  zipCode: string(),
  managerId: string().nullable(),
  profilePictureUrl: string(),
  workerProfile: workerProfileSchema,
  employmentHistory: array().of(employmentHistorySchema),
});

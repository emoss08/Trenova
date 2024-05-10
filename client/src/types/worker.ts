import { type BaseModel } from "@/types/organization";
import { type StatusChoiceProps } from ".";

export interface Worker extends BaseModel {
  id: string;
  code: string;
  status: StatusChoiceProps;
  workerType: string;
  firstName: string;
  lastName: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  fleetCode: string;
  zipCode: string;
  depot?: string | null;
  manager: string;
  profilePicture?: string | null;
  thumbnail?: string;
  enteredBy: string;
  profile: WorkerProfile;

  currentHos?: WorkerHOS | null;
  edges: {
    contacts: WorkerContact[];
    comments: WorkerComment[];
    timeAway: WorkerTimeAway[];
  };
}

export type WorkerFormValues = Omit<
  Worker,
  "organizationId" | "createdAt" | "updatedAt" | "id" | "version" | "edges"
>;

export interface WorkerProfile extends BaseModel {
  worker: string;
  race?: string;
  sex?: string;
  dateOfBirth?: string | null;
  licenseState?: string;
  licenseExpirationDate?: string | null;
  endorsements?: string;
  hazmatExpirationDate?: string | null;
  hm126ExpirationDate?: string | null;
  hireDate?: string | null;
  terminationDate?: string | null;
  reviewDate?: string | null;
  physicalDueDate?: string | null;
  mvrDueDate?: string | null;
  medicalCertDate?: string | null;
}

export interface WorkerContact extends BaseModel {
  id: string;
  worker: string;
  name: string;
  phone?: number | null;
  email?: string;
  relationship?: string;
  isPrimary: boolean;
  mobilePhone?: number | null;
}

export interface WorkerComment extends BaseModel {
  id: string;
  worker: string;
  commentType: string;
  comment: string;
  enteredBy: string;
}

export interface WorkerTimeAway extends BaseModel {
  id: string;
  worker: string;
  startDate: string;
  endDate: string;
  leaveType: string;
}

export interface WorkerHOS extends BaseModel {
  id: string;
  driveTime: number;
  offDutyTime: number;
  sleeperBerthTime: number;
  onDutyTime: number;
  violationTime: number;
  currentStatus: string;
  currentLocation: string;
  seventyHourTime: number;
  milesDriven: number;
  logDate: string;
  lastResetDate: string;
}

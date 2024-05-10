import { type ShipmentStatusChoiceProps } from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";

export interface QualifierCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
}

export type QualifierCodeFormValues = Omit<
  QualifierCode,
  "id" | "organization" | "created" | "modified"
>;

export type StopTypeProps = "P" | "SP" | "SD" | "D" | "DO";

export interface Stop extends BaseModel {
  id: string;
  status: ShipmentStatusChoiceProps;
  sequence?: number | undefined;
  movement: string;
  location?: string;
  pieces?: number;
  weight?: string;
  addressLine?: string;
  appointmentTimeWindowStart: string;
  appointmentTimeWindowEnd: string;
  arrivalTime?: string;
  departureTime?: string;
  stopType: StopTypeProps;
  stopComments?: StopCommentFormValues[];
}

export type StopFormValues = Omit<
  Stop,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "movement" | "version"
>;

export interface StopComment extends BaseModel {
  id: string;
  qualifierCode: string;
  value: string;
}

export type StopCommentFormValues = Omit<
  StopComment,
  "id" | "organization" | "created" | "modified"
>;

export interface ServiceIncident extends BaseModel {
  id: string;
  movement: string;
  stop: string;
  delayCode?: string;
  delayReason?: string;
  delayTime?: any;
}

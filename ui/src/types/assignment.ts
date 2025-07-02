export type TractorAssignment = {
  primaryWorkerId: string;
  secondaryWorkerId?: string;
};

export enum AssignmentStatus {
  New = "New",
  InProgress = "InProgress",
  Completed = "Completed",
  Canceled = "Canceled",
}

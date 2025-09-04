/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export enum WorkerType {
  Employee = "Employee",
  Contractor = "Contractor",
}

export enum ComplianceStatus {
  Compliant = "Compliant",
  NonCompliant = "NonCompliant",
  Pending = "Pending",
}

export enum Endorsement {
  None = "O",
  Tanker = "N",
  Hazmat = "H",
  TankerHazmat = "X",
  Passenger = "P",
  DoublesTriples = "T",
}

// returns value of EndorsementType as Endorsement
export const mapToEndorsement = (endorsement: Endorsement) => {
  const endorsementLabels = {
    O: "None",
    N: "Tanker",
    H: "Hazmat",
    X: "Tanker/Hazmat",
    P: "Passenger",
    T: "Doubles/Triples",
  };

  return endorsementLabels[endorsement];
};

export enum PTOStatus {
  Requested = "Requested",
  Approved = "Approved",
  Rejected = "Rejected",
  Cancelled = "Cancelled",
}

export enum PTOType {
  Personal = "Personal",
  Vacation = "Vacation",
  Sick = "Sick",
  Holiday = "Holiday",
  Bereavement = "Bereavement",
  Maternity = "Maternity",
  Paternity = "Paternity",
}

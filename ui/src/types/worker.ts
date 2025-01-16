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
  Vacation = "Vacation",
  Sick = "Sick",
  Holiday = "Holiday",
  Bereavement = "Bereavement",
  Maternity = "Maternity",
  Paternity = "Paternity",
}

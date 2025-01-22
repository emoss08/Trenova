export enum LocationCategoryType {
  Terminal = "Terminal",
  Warehouse = "Warehouse",
  DistributionCenter = "DistributionCenter",
  TruckStop = "TruckStop",
  RestArea = "RestArea",
  CustomerLocation = "CustomerLocation",
  Port = "Port",
  RailYard = "RailYard",
  MaintenanceFacility = "MaintenanceFacility",
}

export enum FacilityType {
  CrossDock = "CrossDock",
  StorageWarehouse = "StorageWarehouse",
  ColdStorage = "ColdStorage",
  HazmatFacility = "HazmatFacility",
  IntermodalFacility = "IntermodalFacility",
}

// returns value of FacilityType as FacilityType
export const mapToFacilityType = (facilityType: FacilityType) => {
  const facilityTypeLabels = {
    CrossDock: "Cross Dock",
    StorageWarehouse: "Storage Warehouse",
    ColdStorage: "Cold Storage",
    HazmatFacility: "Hazmat Facility",
    IntermodalFacility: "Intermodal Facility",
  };

  return facilityTypeLabels[facilityType];
};

export const mapToLocationCategoryType = (
  locationCategoryType: LocationCategoryType,
) => {
  const locationCategoryTypeLabels = {
    Terminal: "Terminal",
    Warehouse: "Warehouse",
    DistributionCenter: "Distribution Center",
    TruckStop: "Truck Stop",
    RestArea: "Rest Area",
    CustomerLocation: "Customer Location",
    Port: "Port",
    RailYard: "Rail Yard",
    MaintenanceFacility: "Maintenance Facility",
  };

  return locationCategoryTypeLabels[locationCategoryType];
};

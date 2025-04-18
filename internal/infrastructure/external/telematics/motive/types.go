package motive

// -------------------------- Common Types --------------------------
type Address struct {
	Street  string `json:"street,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Zip     string `json:"zip,omitempty"`
	Country string `json:"country,omitempty"`
}

type ExceptionFields struct {
	//nolint:tagliatelle // This is a field name from the API
	Exception24HourRestart bool `json:"exception_24_hour_restart"`
	//nolint:tagliatelle // This is a field name from the API
	Exception8HourBreak bool `json:"exception_8_hour_break"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionWaitTime bool `json:"exception_wait_time"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionShortHaul bool `json:"exception_short_haul"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionCAFarmSchoolBus bool `json:"exception_ca_farm_school_bus"`
}

type VehicleBaseFields struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  string `json:"year"`
	Vin   string `json:"vin,omitempty"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateState string `json:"license_plate_state,omitempty"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateNumber string `json:"license_plate_number,omitempty"`
}

type TimestampFields struct {
	//nolint:tagliatelle // This is a field name from the API
	CreatedAt string `json:"created_at"`
	//nolint:tagliatelle // This is a field name from the API
	UpdatedAt string `json:"updated_at"`
}

type ExternalIDAttributes struct {
	//nolint:tagliatelle // This is a field name from the API
	ExternalID string `json:"external_id"`
	//nolint:tagliatelle // This is a field name from the API
	IntegrationName string `json:"integration_name"`
}

type Pagination struct {
	//nolint:tagliatelle // This is a field name from the API
	PerPage int `json:"per_page"`
	//nolint:tagliatelle // This is a field name from the API
	PageNo int `json:"page_no"`
	Total  int `json:"total"`
}

// -------------------------- Asset endpoints --------------------------
type CreateNewAssetRequest struct {
	// * Required fieldss
	Name  string `json:"name"`
	Type  string `json:"type"`
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  string `json:"year"`

	// * Optional fields
	Vin    string `json:"vin,omitempty"`
	Length int    `json:"length,omitempty"`
	Axle   int    `json:"axle,omitempty"`
	GVWR   int    `json:"gvwr,omitempty"`
	GAWR   int    `json:"gawr,omitempty"`
	//nolint:tagliatelle // This is a field name from the API
	LengthMetricUnits bool `json:"length_metric_units,omitempty"`

	//nolint:tagliatelle // This is a field name from the API
	WeightMetricsUnits bool `json:"weight_metric_units,omitempty"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateState string `json:"license_plate_state,omitempty"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateNumber string `json:"license_plate_number,omitempty"`
	Leased             bool   `json:"leased,omitempty"`
	Notes              string `json:"notes,omitempty"`

	//nolint:tagliatelle // This is a field name from the API
	AssetGatewayID string `json:"asset_gateway_id,omitempty"`

	//nolint:tagliatelle // This is a field name from the API
	ExternalIDsAttrs ExternalIDAttributes `json:"external_ids_attributes"`

	//nolint:tagliatelle // This is a field name from the API
	GroupIDs []int `json:"group_ids,omitempty"`
}

type NewAssetResponse struct {
	Asset struct {
		ID string `json:"id"`
		// * Required fieldss
		Name  string `json:"name"`
		Type  string `json:"type"`
		Make  string `json:"make"`
		Model string `json:"model"`
		Year  string `json:"year"`

		// * Optional fields
		Vin    string `json:"vin,omitempty"`
		Length int    `json:"length,omitempty"`
		Axle   int    `json:"axle,omitempty"`
		GVWR   int    `json:"gvwr,omitempty"`
		GAWR   int    `json:"gawr,omitempty"`
		//nolint:tagliatelle // This is a field name from the API
		LengthMetricUnits bool `json:"length_metric_units,omitempty"`

		//nolint:tagliatelle // This is a field name from the API
		WeightMetricsUnits bool `json:"weight_metric_units,omitempty"`
		//nolint:tagliatelle // This is a field name from the API
		LicensePlateState string `json:"license_plate_state,omitempty"`
		//nolint:tagliatelle // This is a field name from the API
		LicensePlateNumber string `json:"license_plate_number,omitempty"`
		Leased             bool   `json:"leased,omitempty"`
		Notes              string `json:"notes,omitempty"`

		//nolint:tagliatelle // This is a field name from the API
		AssetGatewayID string `json:"asset_gateway_id,omitempty"`

		//nolint:tagliatelle // This is a field name from the API
		ExternalIDsAttrs ExternalIDAttributes `json:"external_ids_attributes"`

		//nolint:tagliatelle // This is a field name from the API
		GroupIDs []int `json:"group_ids,omitempty"`
	} `json:"asset"`
}

// -------------------------- Company endpoints --------------------------
type Company struct {
	ID int `json:"id"`
	//nolint:tagliatelle // This is a field name from the API
	CompanyID string  `json:"company_id"`
	Name      string  `json:"name"`
	Address   Address `json:"address"`
	//nolint:tagliatelle // This is a field name from the API
	DOTIDs []string `json:"dot_ids"`
	Cycle  string   `json:"cycle"`
	//nolint:tagliatelle // This is a field name from the API
	TimeZone string `json:"time_zone"`

	ExceptionFields

	//nolint:tagliatelle // This is a field name from the API
	ExceptionAdverseDriving bool `json:"exception_adverse_driving"`
	//nolint:tagliatelle // This is a field name from the API
	MetricUnits bool `json:"metric_units"`
	//nolint:tagliatelle // This is a field name from the API
	MinuteLogs bool `json:"minute_logs"`
	//nolint:tagliatelle // This is a field name from the API
	SubscriptionPlan string `json:"subscription_plan"`
}

type CompaniesResponse struct {
	Companies  []Company  `json:"companies"`
	Pagination Pagination `json:"pagination"`
}

// -------------------------- Vehicle endpoints --------------------------

type CreateNewVehicleRequest struct {
	Number int `json:"number"`
	//nolint:tagliatelle // This is a field name from the API
	ELDDeviceID int    `json:"eld_device_id"` // * This is a vehicle gateway id
	Status      string `json:"status"`
	IFTA        bool   `json:"ifta"`
	//nolint:tagliatelle // This is a field name from the API
	MetricUnits bool `json:"metric_units"`
	//nolint:tagliatelle // This is a field name from the API
	FuelType string `json:"fuel_type"`
	Vin      string `json:"vin"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateState string `json:"license_plate_state"`
	//nolint:tagliatelle // This is a field name from the API
	LicensePlateNumber string `json:"license_plate_number"`
	Make               string `json:"make"`
	Model              string `json:"model"`
	Year               string `json:"year"`
	//nolint:tagliatelle // This is a field name from the API
	PreventAutoOdometerEntry bool `json:"prevent_auto_odometer_entry"`
	//nolint:tagliatelle // This is a field name from the API
	ExternalIDsAttrs ExternalIDAttributes `json:"external_ids_attributes"`
	//nolint:tagliatelle // This is a field name from the API
	PermanentDriverID int    `json:"permanent_driver_id"`
	Notes             string `json:"notes"`
	//nolint:tagliatelle // This is a field name from the API
	GroupIDs []int `json:"group_ids"`
	//nolint:tagliatelle // This is a field name from the API
	DriverFacingCamera int `json:"driver_facing_camera"`
	//nolint:tagliatelle // This is a field name from the API
	IncabAudioRecording int `json:"incab_audio_recording"`
}

type NewVehicleResponse struct {
	Vehicle struct {
		ID     int `json:"id"`
		Number int `json:"number"`
		//nolint:tagliatelle // This is a field name from the API
		ELDDeviceID int    `json:"eld_device_id"` // * This is a vehicle gateway id
		Status      string `json:"status"`
		IFTA        bool   `json:"ifta"`
		//nolint:tagliatelle // This is a field name from the API
		MetricUnits bool `json:"metric_units"`
		//nolint:tagliatelle // This is a field name from the API
		FuelType string `json:"fuel_type"`
		Vin      string `json:"vin"`
		//nolint:tagliatelle // This is a field name from the API
		LicensePlateState string `json:"license_plate_state"`
		//nolint:tagliatelle // This is a field name from the API
		LicensePlateNumber string `json:"license_plate_number"`
		Make               string `json:"make"`
		Model              string `json:"model"`
		Year               string `json:"year"`
		//nolint:tagliatelle // This is a field name from the API
		PreventAutoOdometerEntry bool `json:"prevent_auto_odometer_entry"`
		//nolint:tagliatelle // This is a field name from the API
		ExternalIDsAttrs ExternalIDAttributes `json:"external_ids_attributes"`
		//nolint:tagliatelle // This is a field name from the API
		PermanentDriverID int    `json:"permanent_driver_id"`
		Notes             string `json:"notes"`
		//nolint:tagliatelle // This is a field name from the API
		GroupIDs []int `json:"group_ids"`
		//nolint:tagliatelle // This is a field name from the API
		DriverFacingCamera int `json:"driver_facing_camera"`
		//nolint:tagliatelle // This is a field name from the API
		IncabAudioRecording int `json:"incab_audio_recording"`
	} `json:"vehicle"`
}

// -------------------------- Vehicle Gateway endpoints --------------------------
type VehicleGateway struct {
	ID         int    `json:"id"`
	Identifier string `json:"identifier"`
	Model      string `json:"model"`
	Vehicle    struct {
		ID     int    `json:"id"`
		Number string `json:"number"`
		Year   string `json:"year"`
		Make   string `json:"make"`
		Model  string `json:"model"`
		Vin    string `json:"vin"`
		//nolint:tagliatelle // This is a field name from the API
		MetricUnits bool `json:"metric_units"`
	} `json:"vehicle"`
}

type VehicleGatewayResponse struct {
	//nolint:tagliatelle // This is a field name from the API
	ELDDevices []VehicleGateway `json:"eld_devices"`
	Pagination Pagination       `json:"pagination"`
}

// -------------------------- User endpoints --------------------------
type NewUserRequest struct {
	Email *string `json:"email"`
	//nolint:tagliatelle // This is a field name from the API
	FirstName string `json:"first_name"`
	//nolint:tagliatelle // This is a field name from the API
	LastName string `json:"last_name"`
	//nolint:tagliatelle // This is a field name from the API
	CompanyReferenceID string `json:"company_reference_id"`
	Phone              string `json:"phone"`
	//nolint:tagliatelle // This is a field name from the API
	PhoneExt *string `json:"phone_ext"`
	//nolint:tagliatelle // This is a field name from the API
	TimeZone string `json:"time_zone"`
	//nolint:tagliatelle // This is a field name from the API
	MetricUnits bool `json:"metric_units"`
	//nolint:tagliatelle // This is a field name from the API
	CarrierName string `json:"carrier_name"`
	//nolint:tagliatelle // This is a field name from the API
	CarrierStreet string `json:"carrier_street"`
	//nolint:tagliatelle // This is a field name from the API
	CarrierCity string `json:"carrier_city"`
	//nolint:tagliatelle // This is a field name from the API
	CarrierState string `json:"carrier_state"`
	//nolint:tagliatelle // This is a field name from the API
	CarrierZip string `json:"carrier_zip"`
	//nolint:tagliatelle // This is a field name from the API
	ViolationAlerts string `json:"violation_alerts"`
	//nolint:tagliatelle // This is a field name from the API
	TerminalStreet string `json:"terminal_street"`
	//nolint:tagliatelle // This is a field name from the API
	TerminalCity string `json:"terminal_city"`
	//nolint:tagliatelle // This is a field name from the API
	TerminalState string `json:"terminal_state"`
	//nolint:tagliatelle // This is a field name from the API
	TerminalZip string `json:"terminal_zip"`
	Cycle       string `json:"cycle"`
	//nolint:tagliatelle // This is a field name from the API
	Exception24HourRestart bool `json:"exception_24_hour_restart"`
	//nolint:tagliatelle // This is a field name from the API
	Exception8HourBreak bool `json:"exception_8_hour_break"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionWaitTime bool `json:"exception_wait_time"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionShortHaul bool `json:"exception_short_haul"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionCAFarmSchoolBus bool    `json:"exception_ca_farm_school_bus"`
	Cycle2                   *string `json:"cycle2"`
	//nolint:tagliatelle // This is a field name from the API
	Exception24HourRestart2 bool `json:"exception_24_hour_restart2"`
	//nolint:tagliatelle // This is a field name from the API
	Exception8HourBreak2 bool `json:"exception_8_hour_break2"`
	//nolint:tagliatelle // This is a field name from the API
	//nolint:tagliatelle // This is a field name from the API
	ExceptionWaitTime2 bool `json:"exception_wait_time2"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionShortHaul2 bool `json:"exception_short_haul2"`
	//nolint:tagliatelle // This is a field name from the API
	ExceptionCAFarmSchoolBus2 bool `json:"exception_ca_farm_school_bus2"`
	//nolint:tagliatelle // This is a field name from the API
	ExportCombined bool `json:"export_combined"`
	//nolint:tagliatelle // This is a field name from the API
	ExportRecap bool `json:"export_recap"`
	//nolint:tagliatelle // This is a field name from the API
	ExportOdometers bool   `json:"export_odometers"`
	Username        string `json:"username"`
	//nolint:tagliatelle // This is a field name from the API
	DriverCompanyID *string `json:"driver_company_id"`
	//nolint:tagliatelle // This is a field name from the API
	MinuteLogs bool `json:"minute_logs"`
	//nolint:tagliatelle // This is a field name from the API
	DutyStatus string `json:"duty_status"`
	//nolint:tagliatelle // This is a field name from the API
	ELDMode string `json:"eld_mode"`
	//nolint:tagliatelle // This is a field name from the API
	DriversLicenseNumber string `json:"drivers_license_number"`
	//nolint:tagliatelle // This is a field name from the API
	DriversLicenseState string `json:"drivers_license_state"`
	//nolint:tagliatelle // This is a field name from the API
	YardMovesEnabled bool `json:"yard_moves_enabled"`
	//nolint:tagliatelle // This is a field name from the API
	PersonalConveyanceEnabled bool   `json:"personal_conveyance_enabled"`
	Role                      string `json:"role"`
	//nolint:tagliatelle // This is a field name from the API
	MobileLastActiveAt *string `json:"mobile_last_active_at"`
	//nolint:tagliatelle // This is a field name from the API
	MobileCurrentSignInAt *string `json:"mobile_current_sign_in_at"`
	//nolint:tagliatelle // This is a field name from the API
	MobileLastSignInAt *string `json:"mobile_last_sign_in_at"`
	//nolint:tagliatelle // This is a field name from the API
	WebLastActiveAt string `json:"web_last_active_at"`
	Status          string `json:"status"`
	//nolint:tagliatelle // This is a field name from the API
	WebCurrentSignInAt string `json:"web_current_sign_in_at"`
	//nolint:tagliatelle // This is a field name from the API
	WebLastSignInAt string `json:"web_last_sign_in_at"`
	//nolint:tagliatelle // This is a field name from the API
	CreatedAt string `json:"created_at"`
	//nolint:tagliatelle // This is a field name from the API
	UpdatedAt string `json:"updated_at"`
	//nolint:tagliatelle // This is a field name from the API
	ExternalIDs []ExternalIDAttributes `json:"external_ids"`
}

type NewUserResponse struct {
	User struct {
		Email *string `json:"email"`
		//nolint:tagliatelle // This is a field name from the API
		FirstName string `json:"first_name"`
		//nolint:tagliatelle // This is a field name from the API
		LastName string `json:"last_name"`
		//nolint:tagliatelle // This is a field name from the API
		CompanyReferenceID string `json:"company_reference_id"`
		Phone              string `json:"phone"`
		//nolint:tagliatelle // This is a field name from the API
		PhoneExt *string `json:"phone_ext"`
		//nolint:tagliatelle // This is a field name from the API
		TimeZone string `json:"time_zone"`
		//nolint:tagliatelle // This is a field name from the API
		MetricUnits bool `json:"metric_units"`
		//nolint:tagliatelle // This is a field name from the API
		CarrierName string `json:"carrier_name"`
		//nolint:tagliatelle // This is a field name from the API
		CarrierStreet string `json:"carrier_street"`
		//nolint:tagliatelle // This is a field name from the API
		CarrierCity string `json:"carrier_city"`
		//nolint:tagliatelle // This is a field name from the API
		CarrierState string `json:"carrier_state"`
		//nolint:tagliatelle // This is a field name from the API
		CarrierZip string `json:"carrier_zip"`
		//nolint:tagliatelle // This is a field name from the API
		ViolationAlerts string `json:"violation_alerts"`
		//nolint:tagliatelle // This is a field name from the API
		TerminalStreet string `json:"terminal_street"`
		//nolint:tagliatelle // This is a field name from the API
		TerminalCity string `json:"terminal_city"`
		//nolint:tagliatelle // This is a field name from the API
		TerminalState string `json:"terminal_state"`
		//nolint:tagliatelle // This is a field name from the API
		TerminalZip string `json:"terminal_zip"`
		Cycle       string `json:"cycle"`
		//nolint:tagliatelle // This is a field name from the API
		Exception24HourRestart bool `json:"exception_24_hour_restart"`
		//nolint:tagliatelle // This is a field name from the API
		Exception8HourBreak bool `json:"exception_8_hour_break"`
		//nolint:tagliatelle // This is a field name from the API
		ExceptionWaitTime bool `json:"exception_wait_time"`
		//nolint:tagliatelle // This is a field name from the API
		ExceptionShortHaul bool `json:"exception_short_haul"`
		//nolint:tagliatelle // This is a field name from the API
		ExceptionCAFarmSchoolBus bool    `json:"exception_ca_farm_school_bus"`
		Cycle2                   *string `json:"cycle2"`
		//nolint:tagliatelle // This is a field name from the API
		Exception24HourRestart2 bool `json:"exception_24_hour_restart2"`
		//nolint:tagliatelle // This is a field name from the API
		Exception8HourBreak2 bool `json:"exception_8_hour_break2"`
		//nolint:tagliatelle // This is a field name from the API
		//nolint:tagliatelle // This is a field name from the API
		ExceptionWaitTime2 bool `json:"exception_wait_time2"`
		//nolint:tagliatelle // This is a field name from the API
		ExceptionShortHaul2 bool `json:"exception_short_haul2"`
		//nolint:tagliatelle // This is a field name from the API
		ExceptionCAFarmSchoolBus2 bool `json:"exception_ca_farm_school_bus2"`
		//nolint:tagliatelle // This is a field name from the API
		ExportCombined bool `json:"export_combined"`
		//nolint:tagliatelle // This is a field name from the API
		ExportRecap bool `json:"export_recap"`
		//nolint:tagliatelle // This is a field name from the API
		ExportOdometers bool   `json:"export_odometers"`
		Username        string `json:"username"`
		//nolint:tagliatelle // This is a field name from the API
		DriverCompanyID *string `json:"driver_company_id"`
		//nolint:tagliatelle // This is a field name from the API
		MinuteLogs bool `json:"minute_logs"`
		//nolint:tagliatelle // This is a field name from the API
		DutyStatus string `json:"duty_status"`
		//nolint:tagliatelle // This is a field name from the API
		ELDMode string `json:"eld_mode"`
		//nolint:tagliatelle // This is a field name from the API
		DriversLicenseNumber string `json:"drivers_license_number"`
		//nolint:tagliatelle // This is a field name from the API
		DriversLicenseState string `json:"drivers_license_state"`
		//nolint:tagliatelle // This is a field name from the API
		YardMovesEnabled bool `json:"yard_moves_enabled"`
		//nolint:tagliatelle // This is a field name from the API
		PersonalConveyanceEnabled bool   `json:"personal_conveyance_enabled"`
		Role                      string `json:"role"`
		//nolint:tagliatelle // This is a field name from the API
		MobileLastActiveAt *string `json:"mobile_last_active_at"`
		//nolint:tagliatelle // This is a field name from the API
		MobileCurrentSignInAt *string `json:"mobile_current_sign_in_at"`
		//nolint:tagliatelle // This is a field name from the API
		MobileLastSignInAt *string `json:"mobile_last_sign_in_at"`
		//nolint:tagliatelle // This is a field name from the API
		WebLastActiveAt string `json:"web_last_active_at"`
		Status          string `json:"status"`
		//nolint:tagliatelle // This is a field name from the API
		WebCurrentSignInAt string `json:"web_current_sign_in_at"`
		//nolint:tagliatelle // This is a field name from the API
		WebLastSignInAt string `json:"web_last_sign_in_at"`
		//nolint:tagliatelle // This is a field name from the API
		CreatedAt string `json:"created_at"`
		//nolint:tagliatelle // This is a field name from the API
		UpdatedAt string `json:"updated_at"`
		//nolint:tagliatelle // This is a field name from the API
		ExternalIDs []ExternalIDAttributes `json:"external_ids"`
	} `json:"user"`
}

type Role string

const (
	Driver       Role = "driver"
	FleetUser    Role = "fleet_user"
	Admin        Role = "admin"
	FleetManager Role = "admin"
)

type ListUsersResponse struct {
	Users []struct {
		User struct {
			Email *string `json:"email"`
			//nolint:tagliatelle // This is a field name from the API
			FirstName string `json:"first_name"`
			//nolint:tagliatelle // This is a field name from the API
			LastName string `json:"last_name"`
			//nolint:tagliatelle // This is a field name from the API
			CompanyReferenceID string `json:"company_reference_id"`
			Phone              string `json:"phone"`
			//nolint:tagliatelle // This is a field name from the API
			PhoneExt *string `json:"phone_ext"`
			//nolint:tagliatelle // This is a field name from the API
			TimeZone string `json:"time_zone"`
			//nolint:tagliatelle // This is a field name from the API
			MetricUnits bool `json:"metric_units"`
			//nolint:tagliatelle // This is a field name from the API
			CarrierName string `json:"carrier_name"`
			//nolint:tagliatelle // This is a field name from the API
			CarrierStreet string `json:"carrier_street"`
			//nolint:tagliatelle // This is a field name from the API
			CarrierCity string `json:"carrier_city"`
			//nolint:tagliatelle // This is a field name from the API
			CarrierState string `json:"carrier_state"`
			//nolint:tagliatelle // This is a field name from the API
			CarrierZip string `json:"carrier_zip"`
			//nolint:tagliatelle // This is a field name from the API
			ViolationAlerts string `json:"violation_alerts"`
			//nolint:tagliatelle // This is a field name from the API
			TerminalStreet string `json:"terminal_street"`
			//nolint:tagliatelle // This is a field name from the API
			TerminalCity string `json:"terminal_city"`
			//nolint:tagliatelle // This is a field name from the API
			TerminalState string `json:"terminal_state"`
			//nolint:tagliatelle // This is a field name from the API
			TerminalZip string `json:"terminal_zip"`
			Cycle       string `json:"cycle"`
			//nolint:tagliatelle // This is a field name from the API
			Exception24HourRestart bool `json:"exception_24_hour_restart"`
			//nolint:tagliatelle // This is a field name from the API
			Exception8HourBreak bool `json:"exception_8_hour_break"`
			//nolint:tagliatelle // This is a field name from the API
			ExceptionWaitTime bool `json:"exception_wait_time"`
			//nolint:tagliatelle // This is a field name from the API
			ExceptionShortHaul bool `json:"exception_short_haul"`
			//nolint:tagliatelle // This is a field name from the API
			ExceptionCAFarmSchoolBus bool    `json:"exception_ca_farm_school_bus"`
			Cycle2                   *string `json:"cycle2"`
			//nolint:tagliatelle // This is a field name from the API
			Exception24HourRestart2 bool `json:"exception_24_hour_restart2"`
			//nolint:tagliatelle // This is a field name from the API
			Exception8HourBreak2 bool `json:"exception_8_hour_break2"`
			//nolint:tagliatelle // This is a field name from the API
			//nolint:tagliatelle // This is a field name from the API
			ExceptionWaitTime2 bool `json:"exception_wait_time2"`
			//nolint:tagliatelle // This is a field name from the API
			ExceptionShortHaul2 bool `json:"exception_short_haul2"`
			//nolint:tagliatelle // This is a field name from the API
			ExceptionCAFarmSchoolBus2 bool `json:"exception_ca_farm_school_bus2"`
			//nolint:tagliatelle // This is a field name from the API
			ExportCombined bool `json:"export_combined"`
			//nolint:tagliatelle // This is a field name from the API
			ExportRecap bool `json:"export_recap"`
			//nolint:tagliatelle // This is a field name from the API
			ExportOdometers bool   `json:"export_odometers"`
			Username        string `json:"username"`
			//nolint:tagliatelle // This is a field name from the API
			DriverCompanyID *string `json:"driver_company_id"`
			//nolint:tagliatelle // This is a field name from the API
			MinuteLogs bool `json:"minute_logs"`
			//nolint:tagliatelle // This is a field name from the API
			DutyStatus string `json:"duty_status"`
			//nolint:tagliatelle // This is a field name from the API
			ELDMode string `json:"eld_mode"`
			//nolint:tagliatelle // This is a field name from the API
			DriversLicenseNumber string `json:"drivers_license_number"`
			//nolint:tagliatelle // This is a field name from the API
			DriversLicenseState string `json:"drivers_license_state"`
			//nolint:tagliatelle // This is a field name from the API
			YardMovesEnabled bool `json:"yard_moves_enabled"`
			//nolint:tagliatelle // This is a field name from the API
			PersonalConveyanceEnabled bool `json:"personal_conveyance_enabled"`

			// * The majority of the time we only want to return the role of driver
			Role string `json:"role"`

			//nolint:tagliatelle // This is a field name from the API
			MobileLastActiveAt *string `json:"mobile_last_active_at"`
			//nolint:tagliatelle // This is a field name from the API
			MobileCurrentSignInAt *string `json:"mobile_current_sign_in_at"`
			//nolint:tagliatelle // This is a field name from the API
			MobileLastSignInAt *string `json:"mobile_last_sign_in_at"`
			//nolint:tagliatelle // This is a field name from the API
			WebLastActiveAt string `json:"web_last_active_at"`
			Status          string `json:"status"`
			//nolint:tagliatelle // This is a field name from the API
			WebCurrentSignInAt string `json:"web_current_sign_in_at"`
			//nolint:tagliatelle // This is a field name from the API
			WebLastSignInAt string `json:"web_last_sign_in_at"`
			//nolint:tagliatelle // This is a field name from the API
			CreatedAt string `json:"created_at"`
			//nolint:tagliatelle // This is a field name from the API
			UpdatedAt string `json:"updated_at"`
			//nolint:tagliatelle // This is a field name from the API
			ExternalIDs []ExternalIDAttributes `json:"external_ids"`
		} `json:"user"`
	} `json:"users"`
	Pagination Pagination `json:"pagination"`
}

// -------------------------- Dispatch Location endpoints --------------------------
type DispatchLocation struct {
	ID int `json:"id"`
	//nolint:tagliatelle // This is a field name from the API
	VendorID string  `json:"vendor_id"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Name     string  `json:"name"`
	Address1 string  `json:"address1"`
	Address2 *string `json:"address2"`
	City     string  `json:"city"`
	State    string  `json:"state"`
	Zip      string  `json:"zip"`
	Country  string  `json:"country"`
	//nolint:tagliatelle // This is a field name from the API
	ArriveRadius int `json:"arrive_radius"`
	//nolint:tagliatelle // This is a field name from the API
	DepartRadius int     `json:"depart_radius"`
	Phone        *string `json:"phone"`
}

type DispatchLocationWrapper struct {
	//nolint:tagliatelle // This is a field name from the API
	DispatchLocation DispatchLocation `json:"dispatch_location"`
}

type DispatchLocationsResponse struct {
	//nolint:tagliatelle // This is a field name from the API
	DispatchLocations []DispatchLocationWrapper `json:"dispatch_locations"`
	Pagination        Pagination                `json:"pagination"`
}

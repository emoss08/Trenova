package google

type RouteAvoidance string

// TODO(Wolfred): We'll need to map these to the actual values used by Google Maps
// and then add them to the API
const (
	// AvoidTolls avoids tolls
	RouteAvoidanceAvoidTolls = RouteAvoidance("AvoidTolls") // `tolls` in google maps

	// AvoidHighways avoids highways
	RouteAvoidanceAvoidHighways = RouteAvoidance("AvoidHighways") // `highways` in google maps

	// AvoidFerries avoids ferries
	RouteAvoidanceAvoidFerries = RouteAvoidance("AvoidFerries") // `ferries` in google maps

	// Indoor avoids indoor routes
	RouteAvoidanceAvoidIndoor = RouteAvoidance("AvoidIndoor") // `indoor` in google maps
)

type RouteModel string

const (
	RouteModelBestGuess   = RouteModel("BestGuess")   // `best_guess` in google maps (Default)
	RouteModelOptimistic  = RouteModel("Optimistic")  // `optimistic` in google maps
	RouteModelPessimistic = RouteModel("Pessimistic") // `pessimistic` in google maps
)

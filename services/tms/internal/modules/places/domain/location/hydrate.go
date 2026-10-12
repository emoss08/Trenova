package location

func HydrateGeofences(entities ...*Location) error {
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		if err := entity.PopulateGeofenceVertices(); err != nil {
			return err
		}
	}

	return nil
}

package shipment

func (s *Shipment) CloneMoveGraph() *Shipment {
	if s == nil {
		return nil
	}

	clone := *s
	clone.Moves = make([]*ShipmentMove, 0, len(s.Moves))

	for _, move := range s.Moves {
		if move == nil {
			clone.Moves = append(clone.Moves, nil)
			continue
		}

		moveClone := *move
		moveClone.Stops = make([]*Stop, 0, len(move.Stops))
		if move.Assignment != nil {
			assignmentClone := *move.Assignment
			moveClone.Assignment = &assignmentClone
		}

		for _, stop := range move.Stops {
			if stop == nil {
				moveClone.Stops = append(moveClone.Stops, nil)
				continue
			}

			stopClone := *stop
			moveClone.Stops = append(moveClone.Stops, &stopClone)
		}

		clone.Moves = append(clone.Moves, &moveClone)
	}

	return &clone
}

package addresses

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCreateRequest(t *testing.T) {
	t.Parallel()

	circle := &GeofenceCircle{RadiusMeters: 150}
	valid := CreateRequest{
		Name:             "Origin DC",
		FormattedAddress: "1 Main St, Springfield, IL",
		Geofence:         Geofence{Circle: circle},
	}
	longNotes := strings.Repeat("n", 281)

	tests := []struct {
		name    string
		mutate  func(*CreateRequest)
		wantErr error
	}{
		{name: "valid", mutate: func(*CreateRequest) {}},
		{
			name:    "missing name",
			mutate:  func(r *CreateRequest) { r.Name = "" },
			wantErr: ErrNameRequired,
		},
		{
			name:    "long name",
			mutate:  func(r *CreateRequest) { r.Name = strings.Repeat("a", 256) },
			wantErr: ErrNameTooLong,
		},
		{
			name:    "missing formatted address",
			mutate:  func(r *CreateRequest) { r.FormattedAddress = " " },
			wantErr: ErrFormattedAddressRequired,
		},
		{
			name:    "long formatted address",
			mutate:  func(r *CreateRequest) { r.FormattedAddress = strings.Repeat("a", 1025) },
			wantErr: ErrFormattedAddressTooLong,
		},
		{
			name:    "long notes",
			mutate:  func(r *CreateRequest) { r.Notes = &longNotes },
			wantErr: ErrNotesTooLong,
		},
		{
			name:    "missing geofence",
			mutate:  func(r *CreateRequest) { r.Geofence = Geofence{} },
			wantErr: ErrGeofenceRequired,
		},
		{
			name: "zero radius",
			mutate: func(r *CreateRequest) {
				r.Geofence = Geofence{Circle: &GeofenceCircle{}}
			},
			wantErr: ErrGeofenceRadiusInvalid,
		},
		{
			name: "both shapes",
			mutate: func(r *CreateRequest) {
				r.Geofence.Polygon = &GeofencePolygon{Vertices: make([]GeofenceVertex, 3)}
			},
			wantErr: ErrGeofenceMutuallyExclusive,
		},
		{
			name: "polygon too small",
			mutate: func(r *CreateRequest) {
				r.Geofence = Geofence{Polygon: &GeofencePolygon{Vertices: make([]GeofenceVertex, 2)}}
			},
			wantErr: ErrGeofencePolygonVerticesBounds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := valid
			req.Geofence = Geofence{Circle: circle}
			tt.mutate(&req)
			err := ValidateCreateRequest(req)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

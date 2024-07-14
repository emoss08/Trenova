package casbin

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_newCasbinPolicy(t *testing.T) {
	type args struct {
		ptype string
		rule  []string
	}
	tests := []struct {
		name string
		args args
		want CasbinPolicy
	}{
		{
			name: "success when ptype is p and one rules is provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
			},
		},
		{
			name: "success when ptype is p and two rules are provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice", "data1"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
				V1:    "data1",
			},
		},
		{
			name: "success when ptype is p and three rules are provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice", "data1", "read"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
				V1:    "data1",
				V2:    "read",
			},
		},
		{
			name: "success when ptype is p and four rules are provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice", "data1", "read", "allow"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
				V1:    "data1",
				V2:    "read",
				V3:    "allow",
			},
		},
		{
			name: "success when ptype is p and five rules are provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice", "data1", "read", "allow", "1"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
				V1:    "data1",
				V2:    "read",
				V3:    "allow",
				V4:    "1",
			},
		},
		{
			name: "success when ptype is p and six rules are provided",
			args: args{
				ptype: "p",
				rule:  []string{"alice", "data1", "read", "allow", "1", "2"},
			},
			want: CasbinPolicy{
				PType: "p",
				V0:    "alice",
				V1:    "data1",
				V2:    "read",
				V3:    "allow",
				V4:    "1",
				V5:    "2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newCasbinPolicy(tt.args.ptype, tt.args.rule)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("newCasbinPolicy() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCasbinPolicy_toSlice(t *testing.T) {
	type fields struct {
		ptype string
		v0    string
		v1    string
		v2    string
		v3    string
		v4    string
		v5    string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "success when ptype is p and one rules is provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
			},
			want: []string{"p", "alice"},
		},
		{
			name: "success when ptype is p and two rules are provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
				v1:    "data1",
			},
			want: []string{"p", "alice", "data1"},
		},
		{
			name: "success when ptype is p and three rules are provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
				v1:    "data1",
				v2:    "read",
			},
			want: []string{"p", "alice", "data1", "read"},
		},
		{
			name: "success when ptype is p and four rules are provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
				v1:    "data1",
				v2:    "read",
				v3:    "allow",
			},
			want: []string{"p", "alice", "data1", "read", "allow"},
		},
		{
			name: "success when ptype is p and five rules are provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
				v1:    "data1",
				v2:    "read",
				v3:    "allow",
				v4:    "1",
			},
			want: []string{"p", "alice", "data1", "read", "allow", "1"},
		},
		{
			name: "success when ptype is p and six rules are provided",
			fields: fields{
				ptype: "p",
				v0:    "alice",
				v1:    "data1",
				v2:    "read",
				v3:    "allow",
				v4:    "1",
				v5:    "2",
			},
			want: []string{"p", "alice", "data1", "read", "allow", "1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := CasbinPolicy{
				PType: tt.fields.ptype,
				V0:    tt.fields.v0,
				V1:    tt.fields.v1,
				V2:    tt.fields.v2,
				V3:    tt.fields.v3,
				V4:    tt.fields.v4,
				V5:    tt.fields.v5,
			}
			if diff := cmp.Diff(tt.want, policy.toSlice()); diff != "" {
				t.Errorf("toSlice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCasbinPolicy_filterValues(t *testing.T) {
	type fields struct {
		v0 string
		v1 string
		v2 string
		v3 string
		v4 string
		v5 string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "success when one rules is provided",
			fields: fields{
				v0: "alice",
			},
			want: []string{"alice"},
		},
		{
			name: "success when two rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
			},
			want: []string{"alice", "data1"},
		},
		{
			name: "success when three rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
			},
			want: []string{"alice", "data1", "read"},
		},
		{
			name: "success when four rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
			},
			want: []string{"alice", "data1", "read", "allow"},
		},
		{
			name: "success when five rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
				v4: "1",
			},
			want: []string{"alice", "data1", "read", "allow", "1"},
		},
		{
			name: "success when six rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
				v4: "1",
				v5: "2",
			},
			want: []string{"alice", "data1", "read", "allow", "1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := CasbinPolicy{
				V0: tt.fields.v0,
				V1: tt.fields.v1,
				V2: tt.fields.v2,
				V3: tt.fields.v3,
				V4: tt.fields.v4,
				V5: tt.fields.v5,
			}
			if diff := cmp.Diff(tt.want, policy.FilterValues()); diff != "" {
				t.Errorf("filterValues() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCasbinPolicy_filterValuesWithKey(t *testing.T) {
	type fields struct {
		v0 string
		v1 string
		v2 string
		v3 string
		v4 string
		v5 string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "success when one rules is provided",
			fields: fields{
				v0: "alice",
			},
			want: map[string]string{"v0": "alice"},
		},
		{
			name: "success when two rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
			},
			want: map[string]string{"v0": "alice", "v1": "data1"},
		},
		{
			name: "success when three rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
			},
			want: map[string]string{"v0": "alice", "v1": "data1", "v2": "read"},
		},
		{
			name: "success when four rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
			},
			want: map[string]string{"v0": "alice", "v1": "data1", "v2": "read", "v3": "allow"},
		},
		{
			name: "success when five rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
				v4: "1",
			},
			want: map[string]string{"v0": "alice", "v1": "data1", "v2": "read", "v3": "allow", "v4": "1"},
		},
		{
			name: "success when six rules are provided",
			fields: fields{
				v0: "alice",
				v1: "data1",
				v2: "read",
				v3: "allow",
				v4: "1",
				v5: "2",
			},
			want: map[string]string{"v0": "alice", "v1": "data1", "v2": "read", "v3": "allow", "v4": "1", "v5": "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := CasbinPolicy{
				V0: tt.fields.v0,
				V1: tt.fields.v1,
				V2: tt.fields.v2,
				V3: tt.fields.v3,
				V4: tt.fields.v4,
				V5: tt.fields.v5,
			}
			if diff := cmp.Diff(tt.want, policy.filterValuesWithKey()); diff != "" {
				t.Errorf("filterValuesWithKey() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

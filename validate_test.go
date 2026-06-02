package openspec

import (
	"strings"
	"testing"
)

func TestValidateSpec(t *testing.T) {
	tests := []struct {
		name string
		spec *Spec
		want string
	}{
		{
			name: "valid",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text:      "The system SHALL issue a token.",
					Scenarios: []Scenario{{RawText: "- WHEN login succeeds\n- THEN a token is returned"}},
				}},
			},
		},
		{
			name: "missing keyword",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text:      "The system issues a token.",
					Scenarios: []Scenario{{RawText: "- WHEN login succeeds\n- THEN a token is returned"}},
				}},
			},
			want: "SHALL or MUST",
		},
		{
			name: "missing scenario",
			spec: &Spec{
				Name:     "auth",
				Overview: "Authentication and session management.",
				Requirements: []Requirement{{
					Text: "The system SHALL issue a token.",
				}},
			},
			want: "scenario",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSpec(tt.spec)
			if tt.want == "" && err != nil {
				t.Fatalf("ValidateSpec() = %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("ValidateSpec() = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestValidateSpecAllowsSoftWarningThresholds(t *testing.T) {
	spec := &Spec{
		Name:     "auth",
		Overview: "short",
		Requirements: []Requirement{{
			Text: "The system SHALL " + strings.Repeat("preserve documented behavior ", 30),
			Scenarios: []Scenario{{
				RawText: "- WHEN behavior is documented\n- THEN validation accepts the spec",
			}},
		}},
	}
	if err := ValidateSpec(spec); err != nil {
		t.Fatal(err)
	}
}

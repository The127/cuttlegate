package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestFlagValidate(t *testing.T) {
	tests := []struct {
		name      string
		flag      Flag
		wantErr   bool
		wantField string
	}{
		{
			name: "valid_bool_flag",
			flag: Flag{
				Key:               "my-flag",
				Name:              "My Flag",
				Type:              FlagTypeBool,
				Variants:          []Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}},
				DefaultVariantKey: "false",
			},
			wantErr: false,
		},
		{
			name: "valid_string_flag",
			flag: Flag{
				Key:               "color-scheme",
				Name:              "Color Scheme",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "blue", Name: "Blue"}, {Key: "green", Name: "Green"}, {Key: "red", Name: "Red"}},
				DefaultVariantKey: "blue",
			},
			wantErr: false,
		},
		{
			name: "empty_name",
			flag: Flag{
				Key:               "my-flag",
				Name:              "",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr: true,
		},
		{
			name: "bool_flag_one_variant",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeBool,
				Variants:          []Variant{{Key: "true", Name: "On"}},
				DefaultVariantKey: "true",
			},
			wantErr:   true,
			wantField: "variants",
		},
		{
			name: "bool_flag_three_variants",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeBool,
				Variants:          []Variant{{Key: "true", Name: "On"}, {Key: "false", Name: "Off"}, {Key: "maybe", Name: "Maybe"}},
				DefaultVariantKey: "true",
			},
			wantErr:   true,
			wantField: "variants",
		},
		{
			name: "bool_flag_wrong_variant_keys",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeBool,
				Variants:          []Variant{{Key: "on", Name: "On"}, {Key: "off", Name: "Off"}},
				DefaultVariantKey: "on",
			},
			wantErr:   true,
			wantField: "variants",
		},
		{
			name: "empty_variants",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeString,
				Variants:          []Variant{},
				DefaultVariantKey: "",
			},
			wantErr:   true,
			wantField: "variants",
		},
		{
			name: "duplicate_variant_keys",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}, {Key: "a", Name: "A2"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "variants",
		},
		{
			name: "default_variant_key_not_in_variants",
			flag: Flag{
				Key:               "my-flag",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}, {Key: "b", Name: "B"}},
				DefaultVariantKey: "c",
			},
			wantErr:   true,
			wantField: "default_variant_key",
		},
		{
			name: "invalid_type",
			flag: Flag{
				Key:               "my-flag",
				Type:              "banana",
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "type",
		},
		{
			name: "empty_type",
			flag: Flag{
				Key:               "my-flag",
				Type:              "",
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "type",
		},
		{
			name: "key_with_space",
			flag: Flag{
				Key:               "my flag",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "key",
		},
		{
			name: "key_starting_with_hyphen",
			flag: Flag{
				Key:               "-my-flag",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "key",
		},
		{
			name: "empty_key",
			flag: Flag{
				Key:               "",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "key",
		},
		{
			name: "key_too_long",
			flag: Flag{
				Key:               strings.Repeat("a", 129),
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr:   true,
			wantField: "key",
		},
		{
			name: "key_exactly_max_length",
			flag: Flag{
				Key:               strings.Repeat("a", MaxKeyLength),
				Name:              "Valid Flag",
				Type:              FlagTypeString,
				Variants:          []Variant{{Key: "a", Name: "A"}},
				DefaultVariantKey: "a",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flag.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Validate() error type = %T, want *ValidationError", err)
				} else if tt.wantField != "" && valErr.Field != tt.wantField {
					t.Errorf("Validate() error field = %q, want %q", valErr.Field, tt.wantField)
				}
			}
		})
	}
}

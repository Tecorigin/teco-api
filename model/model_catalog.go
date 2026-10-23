package model

import (
	"fmt"
	"strings"
	"time"
)

type ModelCatalog struct {
	MaxInputTokens       *int     `json:"max_input_tokens,omitempty"`
	MaxOutputTokens      *int     `json:"max_output_tokens,omitempty"`
	ContextLength        *int     `json:"context_length,omitempty"`
	RPM                  *int     `json:"rpm,omitempty"`
	TPM                  *int     `json:"tpm,omitempty"`
	ShowActualRateLimits bool     `json:"show_actual_rate_limits"`
	InputModalities      []string `json:"input_modalities,omitempty" gorm:"serializer:json;type:text"`
	OutputModalities     []string `json:"output_modalities,omitempty" gorm:"serializer:json;type:text"`
	FilingRegion         string   `json:"filing_region,omitempty" gorm:"type:text"`
	FilingModelName      string   `json:"filing_model_name,omitempty" gorm:"type:text"`
	FilingOrganization   string   `json:"filing_organization,omitempty" gorm:"type:text"`
	FilingNumber         string   `json:"filing_number,omitempty" gorm:"type:text"`
	FilingDate           string   `json:"filing_date,omitempty" gorm:"type:text"`
}

func (catalog *ModelCatalog) Validate() error {
	for _, value := range []*string{
		&catalog.FilingRegion, &catalog.FilingModelName, &catalog.FilingOrganization,
		&catalog.FilingNumber, &catalog.FilingDate,
	} {
		*value = strings.TrimSpace(*value)
	}
	if catalog.FilingDate != "" {
		if _, err := time.Parse("2006-01-02", catalog.FilingDate); err != nil {
			return fmt.Errorf("filing_date must be a valid date in YYYY-MM-DD format")
		}
	}
	for _, field := range []struct {
		name  string
		value *int
		min   int
	}{
		{"max_input_tokens", catalog.MaxInputTokens, 1},
		{"max_output_tokens", catalog.MaxOutputTokens, 1},
		{"context_length", catalog.ContextLength, 1},
		{"rpm", catalog.RPM, 0},
		{"tpm", catalog.TPM, 0},
	} {
		if field.value != nil && (*field.value < field.min || *field.value > 2147483647) {
			return fmt.Errorf("%s must be between %d and 2147483647", field.name, field.min)
		}
	}
	for _, modalities := range [][]string{catalog.InputModalities, catalog.OutputModalities} {
		seen := make(map[string]bool)
		for _, modality := range modalities {
			switch modality {
			case "text", "image", "audio", "video":
			default:
				return fmt.Errorf("unsupported modality: %s", modality)
			}
			if seen[modality] {
				return fmt.Errorf("duplicate modality: %s", modality)
			}
			seen[modality] = true
		}
	}
	return nil
}

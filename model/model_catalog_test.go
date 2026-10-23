package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestModelCatalogPersistenceAndClearing(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB := DB
	DB = database
	t.Cleanup(func() {
		DB = previousDB
		_ = sqlDB.Close()
	})
	require.NoError(t, DB.AutoMigrate(&Model{}))
	input, output, context, rpm, tpm := 32000, 8000, 40000, 0, 100000
	entry := Model{ModelName: "catalog-test", Status: 1, ModelCatalog: ModelCatalog{
		MaxInputTokens: &input, MaxOutputTokens: &output, ContextLength: &context,
		RPM: &rpm, TPM: &tpm, ShowActualRateLimits: true,
		InputModalities: []string{"text", "image"}, OutputModalities: []string{"audio"},
		FilingRegion: "Shanghai", FilingModelName: "Filed model", FilingOrganization: "Example organization",
		FilingNumber: "FILING-2026-001", FilingDate: "2026-10-23",
	}}
	require.NoError(t, entry.Insert())
	var stored Model
	require.NoError(t, DB.First(&stored, entry.Id).Error)
	assert.Equal(t, entry.ModelCatalog, stored.ModelCatalog)
	stored.ModelCatalog = ModelCatalog{}
	require.NoError(t, stored.Update())
	var cleared Model
	require.NoError(t, DB.First(&cleared, entry.Id).Error)
	assert.Equal(t, ModelCatalog{}, cleared.ModelCatalog)
}

func TestModelCatalogFilingNormalization(t *testing.T) {
	catalog := ModelCatalog{
		FilingRegion: "  Shanghai  ", FilingModelName: "  Filed model  ",
		FilingOrganization: "  Example organization  ", FilingNumber: " \t ", FilingDate: " 2026-10-23 ",
	}
	require.NoError(t, catalog.Validate())
	assert.Equal(t, "Shanghai", catalog.FilingRegion)
	assert.Equal(t, "Filed model", catalog.FilingModelName)
	assert.Equal(t, "Example organization", catalog.FilingOrganization)
	assert.Empty(t, catalog.FilingNumber)
	assert.Equal(t, "2026-10-23", catalog.FilingDate)
	catalog = ModelCatalog{FilingDate: " \t "}
	require.NoError(t, catalog.Validate())
	assert.Empty(t, catalog.FilingDate)
}

func TestModelCatalogValidation(t *testing.T) {
	negative, zero, oversized := -1, 0, int(int64(2147483648))
	for _, test := range []struct {
		name    string
		catalog ModelCatalog
		valid   bool
	}{
		{"empty", ModelCatalog{}, true},
		{"unlimited", ModelCatalog{RPM: &zero, TPM: &zero}, true},
		{"zero context", ModelCatalog{ContextLength: &zero}, false},
		{"negative input", ModelCatalog{MaxInputTokens: &negative}, false},
		{"oversized output", ModelCatalog{MaxOutputTokens: &oversized}, false},
		{"negative rpm", ModelCatalog{RPM: &negative}, false},
		{"oversized tpm", ModelCatalog{TPM: &oversized}, false},
		{"unknown input", ModelCatalog{InputModalities: []string{"unknown"}}, false},
		{"duplicate output", ModelCatalog{OutputModalities: []string{"text", "text"}}, false},
		{"filing invalid date", ModelCatalog{FilingDate: "2026-02-30"}, false},
		{"filing invalid date format", ModelCatalog{FilingDate: "23/10/2026"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.valid {
				assert.NoError(t, test.catalog.Validate())
			} else {
				assert.Error(t, test.catalog.Validate())
			}
		})
	}
}

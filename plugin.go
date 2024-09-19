package gorm_migrate_tracker

import (
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
)

// SchemaVersion represents a version of the database schema
type SchemaVersion struct {
	ID        uint      `gorm:"primaryKey"`
	Version   string    `gorm:"uniqueIndex"`
	AppliedAt time.Time
	Changes   string
}

// AutoMigratePlugin is a GORM plugin for tracking AutoMigrate changes
type AutoMigratePlugin struct{}

// Name returns the name of the plugin
func (p *AutoMigratePlugin) Name() string {
	return "AutoMigratePlugin"
}

// Initialize implements the GORM plugin interface
func (p *AutoMigratePlugin) Initialize(db *gorm.DB) error {
	// Ensure the schema version table exists
	err := db.AutoMigrate(&SchemaVersion{})
	if err != nil {
		return fmt.Errorf("failed to create schema version table: %w", err)
	}

	// Replace the default AutoMigrate method
	db.Callback().Create().Replace("gorm:auto_migrate", p.autoMigrateAndTrack)

	return nil
}

// autoMigrateAndTrack performs AutoMigrate and tracks the changes
func (p *AutoMigratePlugin) autoMigrateAndTrack(db *gorm.DB) {
	if db.Statement.Schema == nil || db.Statement.Schema.ModelType == nil {
		return
	}

	// Generate a new version
	version := time.Now().Format("20060102150405")

	// Perform AutoMigrate
	err := db.Migrator().AutoMigrate(db.Statement.Schema.ModelType)
	if err != nil {
		db.AddError(fmt.Errorf("auto migration failed: %w", err))
		return
	}

	// Track changes
	changes := p.generateChangeLog(db.Statement.Schema.ModelType)

	// Record the migration
	schemaVersion := SchemaVersion{
		Version:   version,
		AppliedAt: time.Now(),
		Changes:   changes,
	}

	if err := db.Create(&schemaVersion).Error; err != nil {
		db.AddError(fmt.Errorf("failed to record schema version: %w", err))
	}
}

// generateChangeLog creates a simple change log based on the model
func (p *AutoMigratePlugin) generateChangeLog(model interface{}) string {
	modelType := reflect.TypeOf(model)
	return fmt.Sprintf("AutoMigrated %s", modelType.Name())
}

// GetMigrationHistory retrieves the history of schema changes
func GetMigrationHistory(db *gorm.DB) ([]SchemaVersion, error) {
	var history []SchemaVersion
	if err := db.Order("applied_at desc").Find(&history).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve migration history: %w", err)
	}
	return history, nil
}

package gorm_migrate_tracker

import (
	"fmt"
	"log"
	"os"
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
type AutoMigratePlugin struct {
	Logger *log.Logger
}

// NewAutoMigratePlugin creates a new instance of AutoMigratePlugin with a default logger
func NewAutoMigratePlugin() *AutoMigratePlugin {
	return &AutoMigratePlugin{
		Logger: log.New(os.Stdout, "[AutoMigratePlugin] ", log.LstdFlags),
	}
}

// Name returns the name of the plugin
func (p *AutoMigratePlugin) Name() string {
	p.Logger.Println("Name method called")
	return "AutoMigratePlugin"
}

// Initialize implements the GORM plugin interface
func (p *AutoMigratePlugin) Initialize(db *gorm.DB) error {
	p.Logger.Println("Initialize method called")

	// Ensure the schema version table exists
	p.Logger.Println("Attempting to create SchemaVersion table")
	err := db.AutoMigrate(&SchemaVersion{})
	if err != nil {
		p.Logger.Printf("Failed to create schema version table: %v", err)
		return fmt.Errorf("failed to create schema version table: %w", err)
	}
	p.Logger.Println("SchemaVersion table created or already exists")

	// Register callbacks
	p.Logger.Println("Registering before_auto_migrate callback")
	err = db.Callback().Migrator().Register("automigrate_plugin:before_auto_migrate", p.beforeAutoMigrate)
	if err != nil {
		p.Logger.Printf("Failed to register before_auto_migrate callback: %v", err)
		return fmt.Errorf("failed to register before_auto_migrate callback: %w", err)
	}

	p.Logger.Println("Registering after_auto_migrate callback")
	err = db.Callback().Migrator().Register("automigrate_plugin:after_auto_migrate", p.afterAutoMigrate)
	if err != nil {
		p.Logger.Printf("Failed to register after_auto_migrate callback: %v", err)
		return fmt.Errorf("failed to register after_auto_migrate callback: %w", err)
	}

	p.Logger.Println("Initialize method completed successfully")
	return nil
}

// beforeAutoMigrate is called before AutoMigrate
func (p *AutoMigratePlugin) beforeAutoMigrate(db *gorm.DB) {
	p.Logger.Println("beforeAutoMigrate callback triggered")
	startTime := time.Now()
	db.InstanceSet("automigrate_plugin:start_time", startTime)
	p.Logger.Printf("Set start time: %v", startTime)
}

// afterAutoMigrate is called after AutoMigrate
func (p *AutoMigratePlugin) afterAutoMigrate(db *gorm.DB) {
	p.Logger.Println("afterAutoMigrate callback triggered")

	startTime, ok := db.InstanceGet("automigrate_plugin:start_time")
	if !ok {
		p.Logger.Println("Error: start time not found")
		db.AddError(fmt.Errorf("start time not found"))
		return
	}
	p.Logger.Printf("Retrieved start time: %v", startTime)

	// Generate a new version
	version := startTime.(time.Time).Format("20060102150405")
	p.Logger.Printf("Generated version: %s", version)

	// Track changes
	changes := p.generateChangeLog(db)
	p.Logger.Printf("Generated change log: %s", changes)

	// Record the migration
	schemaVersion := SchemaVersion{
		Version:   version,
		AppliedAt: time.Now(),
		Changes:   changes,
	}

	p.Logger.Println("Attempting to create new SchemaVersion record")
	if err := db.Create(&schemaVersion).Error; err != nil {
		p.Logger.Printf("Failed to record schema version: %v", err)
		db.AddError(fmt.Errorf("failed to record schema version: %w", err))
	} else {
		p.Logger.Println("Successfully created new SchemaVersion record")
	}
}

// generateChangeLog creates a change log based on the migrated models
func (p *AutoMigratePlugin) generateChangeLog(db *gorm.DB) string {
	p.Logger.Println("generateChangeLog method called")

	var changes string
	if models, ok := db.Get("gorm:auto_migrate_models"); ok {
		p.Logger.Println("Retrieved auto_migrate_models from db")
		modelSlice, ok := models.([]interface{})
		if !ok {
			p.Logger.Println("Error: models is not a slice of interface{}")
			return "Unable to determine migrated models"
		}
		for _, model := range modelSlice {
			modelName := reflect.TypeOf(model).Name()
			p.Logger.Printf("AutoMigrated model: %s", modelName)
			changes += fmt.Sprintf("AutoMigrated %s\n", modelName)
		}
	} else {
		p.Logger.Println("No specific models found in db")
		changes = "No specific models found, general AutoMigrate performed"
	}

	p.Logger.Printf("Final change log: %s", changes)
	return changes
}

// GetMigrationHistory retrieves the history of schema changes
func GetMigrationHistory(db *gorm.DB) ([]SchemaVersion, error) {
	log.Println("GetMigrationHistory function called")

	var history []SchemaVersion
	result := db.Order("applied_at desc").Find(&history)
	if result.Error != nil {
		log.Printf("Failed to retrieve migration history: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve migration history: %w", result.Error)
	}

	log.Printf("Retrieved %d migration history records", len(history))
	return history, nil
}


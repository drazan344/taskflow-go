package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel contains common columns for all models
type BaseModel struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TenantModel contains common columns for tenant-scoped models
type TenantModel struct {
	BaseModel
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
}

// BeforeCreate hook for BaseModel to generate UUID
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}

// TableName prefix for tenant-specific tables
func GetTenantTableName(tenantID uuid.UUID, tableName string) string {
	return tableName // For shared schema approach, we use the same table with tenant_id
}

// Common query helpers
type QueryOptions struct {
	Limit  int
	Offset int
	Sort   string
	Order  string
}

// DefaultQueryOptions returns default pagination options
func DefaultQueryOptions() *QueryOptions {
	return &QueryOptions{
		Limit: 20,
		Offset: 0,
		Sort:  "created_at",
		Order: "desc",
	}
}

// ApplyQueryOptions applies pagination and sorting to a GORM query
func ApplyQueryOptions(db *gorm.DB, opts *QueryOptions) *gorm.DB {
	if opts == nil {
		opts = DefaultQueryOptions()
	}

	if opts.Limit > 0 {
		db = db.Limit(opts.Limit)
	}
	
	if opts.Offset > 0 {
		db = db.Offset(opts.Offset)
	}

	if opts.Sort != "" {
		order := "ASC"
		if opts.Order != "" && (opts.Order == "desc" || opts.Order == "DESC") {
			order = "DESC"
		}
		db = db.Order(opts.Sort + " " + order)
	}

	return db
}
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/config"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*gorm.DB, *redis.Client, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DatabaseDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DatabaseDSN)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 20; attempt++ {
		db, err = gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil && sqlDB.PingContext(ctx) == nil {
				break
			}
			if dbErr != nil {
				err = dbErr
			} else {
				err = sqlDB.PingContext(ctx)
			}
		}
		log.Warn("database not ready", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, nil, err
	}
	if err := Seed(ctx, db); err != nil {
		return nil, nil, err
	}
	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, nil, fmt.Errorf("connect redis: %w", err)
		}
	}
	return db, redisClient, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{}, &model.AuditLog{},
		&model.WasteGenerator{},
		&model.CarrierProfile{},
		&model.TransferManifest{},
		&model.ComplianceCheck{},
	)
}

func Seed(ctx context.Context, db *gorm.DB) error {
	var users int64
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&users).Error; err != nil {
		return err
	}
	if users == 0 {
		password, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		seedUsers := []model.User{
			{Username: "admin", DisplayName: "系统管理员", PasswordHash: string(password), Role: model.RoleAdmin, Active: true},
			{Username: "reviewer", DisplayName: "质量复核员", PasswordHash: string(password), Role: model.RoleReviewer, Active: true},
			{Username: "operator", DisplayName: "现场操作员", PasswordHash: string(password), Role: model.RoleOperator, Active: true},
			{Username: "viewer", DisplayName: "只读观察员", PasswordHash: string(password), Role: model.RoleViewer, Active: true},
		}
		if err := db.WithContext(ctx).Create(&seedUsers).Error; err != nil {
			return err
		}
	}

	if err := seedWasteGenerator(ctx, db); err != nil {
		return err
	}

	if err := seedCarrierProfile(ctx, db); err != nil {
		return err
	}

	if err := seedTransferManifest(ctx, db); err != nil {
		return err
	}

	if err := seedComplianceCheck(ctx, db); err != nil {
		return err
	}

	return nil
}

func seedWasteGenerator(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.WasteGenerator{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.WasteGenerator{

		{BaseModel: model.BaseModel{Code: "WG-001", Name: "产废单位示例一", Status: "active", Version: 1,
			Description: "用于启动验证和主要流程演示的产废单位记录"}, PermitNumber: "PERMIT-WG-001", PermitExpiresAt: now.AddDate(1, 0, 0), WasteCategories: "HW08 废矿物油",
			Facility: "危险废物转运合规核验区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-01"},

		{BaseModel: model.BaseModel{Code: "WG-002", Name: "产废单位示例二", Status: "restricted", Version: 1,
			Description: "用于启动验证和主要流程演示的产废单位记录"}, PermitNumber: "PERMIT-WG-002", PermitExpiresAt: now.AddDate(0, 8, 0), WasteCategories: "HW17 表面处理废物",
			Facility: "危险废物转运合规核验区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-02"},

		{BaseModel: model.BaseModel{Code: "WG-003", Name: "产废单位示例三", Status: "suspended", Version: 1,
			Description: "用于启动验证和主要流程演示的产废单位记录"}, PermitNumber: "PERMIT-WG-003", PermitExpiresAt: now.AddDate(0, 5, 0), WasteCategories: "HW49 其他废物",
			Facility: "危险废物转运合规核验区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedCarrierProfile(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.CarrierProfile{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.CarrierProfile{

		{BaseModel: model.BaseModel{Code: "CP-001", Name: "承运资质示例一", Status: "pending", Version: 1,
			Description: "用于启动验证和主要流程演示的承运资质记录"}, LicenseNumber: "CARRIER-LIC-001", LicenseExpiresAt: now.AddDate(1, 0, 0), VehicleCount: 8,
			Facility: "危险废物转运合规核验区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-01"},

		{BaseModel: model.BaseModel{Code: "CP-002", Name: "承运资质示例二", Status: "verified", Version: 1,
			Description: "用于启动验证和主要流程演示的承运资质记录"}, LicenseNumber: "CARRIER-LIC-002", LicenseExpiresAt: now.AddDate(0, 10, 0), VehicleCount: 16,
			Facility: "危险废物转运合规核验区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-02"},

		{BaseModel: model.BaseModel{Code: "CP-003", Name: "承运资质示例三", Status: "restricted", Version: 1,
			Description: "用于启动验证和主要流程演示的承运资质记录"}, LicenseNumber: "CARRIER-LIC-003", LicenseExpiresAt: now.AddDate(0, 4, 0), VehicleCount: 4,
			Facility: "危险废物转运合规核验区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedTransferManifest(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.TransferManifest{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.TransferManifest{

		{BaseModel: model.BaseModel{Code: "TM-001", Name: "转运清单示例一", Status: "draft", Version: 1,
			Description: "用于启动验证和主要流程演示的转运清单记录"}, GeneratorCode: "WG-001", CarrierCode: "CP-002", WasteCode: "HW08-900-249-08", QuantityKg: 1200, Destination: "合规处置中心 A",
			Facility: "危险废物转运合规核验区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-01"},

		{BaseModel: model.BaseModel{Code: "TM-002", Name: "转运清单示例二", Status: "submitted", Version: 1,
			Description: "用于启动验证和主要流程演示的转运清单记录"}, GeneratorCode: "WG-001", CarrierCode: "CP-002", WasteCode: "HW17-336-064-17", QuantityKg: 850, Destination: "资源化利用中心 B",
			Facility: "危险废物转运合规核验区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-02"},

		{BaseModel: model.BaseModel{Code: "TM-003", Name: "转运清单示例三", Status: "in_transit", Version: 1,
			Description: "用于启动验证和主要流程演示的转运清单记录"}, GeneratorCode: "WG-001", CarrierCode: "CP-002", WasteCode: "HW49-900-041-49", QuantityKg: 420, Destination: "安全填埋中心 C",
			Facility: "危险废物转运合规核验区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedComplianceCheck(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.ComplianceCheck{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.ComplianceCheck{

		{BaseModel: model.BaseModel{Code: "CC-001", Name: "合规核验示例一", Status: "pending", Version: 1,
			Description: "用于启动验证和主要流程演示的合规核验记录"}, ManifestCode: "TM-002", Checklist: "产废许可、承运资质、联单数量、处置去向", DecisionBasis: "",
			Facility: "危险废物转运合规核验区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-01"},

		{BaseModel: model.BaseModel{Code: "CC-002", Name: "合规核验示例二", Status: "pass", Version: 1,
			Description: "用于启动验证和主要流程演示的合规核验记录"}, ManifestCode: "TM-002", Checklist: "产废许可、承运资质、联单数量、处置去向", DecisionBasis: "证据齐全且资质有效",
			Facility: "危险废物转运合规核验区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-02"},

		{BaseModel: model.BaseModel{Code: "CC-003", Name: "合规核验示例三", Status: "fail", Version: 1,
			Description: "用于启动验证和主要流程演示的合规核验记录"}, ManifestCode: "TM-003", Checklist: "产废许可、承运资质、联单数量、处置去向", DecisionBasis: "重量凭证存在差异",
			Facility: "危险废物转运合规核验区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-518-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

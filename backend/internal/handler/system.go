package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/service"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SystemHandler struct {
	security         service.SecurityService
	wasteGenerator   service.WasteGeneratorService
	carrierProfile   service.CarrierProfileService
	transferManifest service.TransferManifestService
	complianceCheck  service.ComplianceCheckService
	db               *gorm.DB
	redis            *redis.Client
}

func NewSystemHandler(security service.SecurityService, wasteGenerator service.WasteGeneratorService, carrierProfile service.CarrierProfileService, transferManifest service.TransferManifestService, complianceCheck service.ComplianceCheckService, db *gorm.DB, redisClient *redis.Client) *SystemHandler {
	return &SystemHandler{security: security, wasteGenerator: wasteGenerator, carrierProfile: carrierProfile, transferManifest: transferManifest, complianceCheck: complianceCheck, db: db, redis: redisClient}
}

func (h *SystemHandler) Login(c *gin.Context) {
	var input dto.LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		util.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	response, err := h.security.Login(c.Request.Context(), input)
	if err != nil {
		util.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid username or password")
		return
	}
	util.OK(c, response)
}

func (h *SystemHandler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		util.Fail(c, http.StatusServiceUnavailable, "database_unavailable", "database ping failed")
		return
	}
	redisState := "disabled"
	if h.redis != nil {
		if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
			util.Fail(c, http.StatusServiceUnavailable, "redis_unavailable", "redis ping failed")
			return
		}
		redisState = "ready"
	}
	util.OK(c, gin.H{"status": "ok", "database": "ready", "redis": redisState})
}

func (h *SystemHandler) Overview(c *gin.Context) {
	ctx := c.Request.Context()
	result := make(map[string]any)

	wasteGeneratorCounts, err := h.wasteGenerator.StatusCounts(ctx)
	if err != nil {
		handleError(c, err)
		return
	}
	result["generators"] = wasteGeneratorCounts

	carrierProfileCounts, err := h.carrierProfile.StatusCounts(ctx)
	if err != nil {
		handleError(c, err)
		return
	}
	result["carriers"] = carrierProfileCounts

	transferManifestCounts, err := h.transferManifest.StatusCounts(ctx)
	if err != nil {
		handleError(c, err)
		return
	}
	result["manifests"] = transferManifestCounts

	complianceCheckCounts, err := h.complianceCheck.StatusCounts(ctx)
	if err != nil {
		handleError(c, err)
		return
	}
	result["checks"] = complianceCheckCounts

	util.OK(c, result)
}

func (h *SystemHandler) Audits(c *gin.Context) {
	query := bindPage(c)
	logs, total, err := h.security.ListAudits(c.Request.Context(), query.Page, query.PageSize, query.Search)
	if err != nil {
		handleError(c, err)
		return
	}
	util.Page(c, logs, query.Page, query.PageSize, total)
}

func (h *SystemHandler) Session(c *gin.Context) {
	displayName, _ := c.Get("displayName")
	util.OK(c, dto.SessionResponse{
		Username: actorFromContext(c), DisplayName: strings.TrimSpace(stringValue(displayName)),
		Role: roleFromContext(c), RequestID: requestIDFromContext(c),
	})
}

func (h *SystemHandler) Runtime(c *gin.Context) {
	util.OK(c, h.security.RuntimeConfig())
}

func (h *SystemHandler) AuditSummary(c *gin.Context) {
	var query dto.AuditSummaryQuery
	_ = c.ShouldBindQuery(&query)
	summary, err := h.security.AuditSummary(c.Request.Context(), query.Window())
	if err != nil {
		handleError(c, err)
		return
	}
	util.OK(c, summary)
}

func (h *SystemHandler) EntityHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	entityType := strings.TrimSpace(c.Param("entityType"))
	if entityType == "" || len(entityType) > 80 {
		util.Fail(c, http.StatusBadRequest, "invalid_entity_type", "entityType must be 1-80 characters")
		return
	}
	var query dto.EntityHistoryQuery
	_ = c.ShouldBindQuery(&query)
	logs, err := h.security.EntityHistory(c.Request.Context(), entityType, id, query.NormalizedLimit())
	if err != nil {
		handleError(c, err)
		return
	}
	util.OK(c, logs)
}

func bindPage(c *gin.Context) dto.PageQuery {
	var query dto.PageQuery
	_ = c.ShouldBindQuery(&query)
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	return query
}

func parseID(c *gin.Context) (uint, bool) {
	raw, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || raw == 0 {
		util.Fail(c, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
		return 0, false
	}
	return uint(raw), true
}

func actorFromContext(c *gin.Context) string {
	value, exists := c.Get("username")
	if !exists {
		return "anonymous"
	}
	return strings.TrimSpace(value.(string))
}

func roleFromContext(c *gin.Context) string {
	value, exists := c.Get("role")
	if !exists {
		return "viewer"
	}
	return strings.TrimSpace(value.(string))
}

func requestIDFromContext(c *gin.Context) string {
	value, exists := c.Get("requestID")
	if !exists {
		return c.Writer.Header().Get("X-Request-ID")
	}
	return value.(string)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

package handlers

import (
	"net/http"

	"github.com/ctwj/urldb/services"
	"github.com/gin-gonic/gin"
)

// ComplianceHandler serves the administrator-only compliance audit report and
// aggregate operations dashboard.
type ComplianceHandler struct {
	service *services.ComplianceDashboardService
}

func NewComplianceHandler(service *services.ComplianceDashboardService) *ComplianceHandler {
	return &ComplianceHandler{service: service}
}

func (h *ComplianceHandler) GetDashboard(c *gin.Context) {
	dashboard, err := h.service.GetDashboard()
	if err != nil {
		ErrorResponse(c, "获取合规审计看板失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	SuccessResponse(c, dashboard)
}

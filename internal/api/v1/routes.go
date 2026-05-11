package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/namejlt/AgentFlowPro/internal/api/middleware"
	"github.com/namejlt/AgentFlowPro/internal/app"
)

func Register(g *gin.RouterGroup, a *app.App) {
	// users admin
	ug := g.Group("")
	ug.Use(middleware.RequireRole("admin"))
	{
		ug.POST("/users", a.CreateUser)
		ug.GET("/users", a.ListUsers)
		ug.GET("/users/:id", a.GetUser)
		ug.PATCH("/users/:id", a.PatchUser)
	}

	// models / llm
	g.POST("/models", middleware.RequireRole("creator", "admin"), a.CreateModel)
	g.PUT("/models/:id", middleware.RequireRole("creator", "admin"), a.UpdateModel)
	g.DELETE("/models/:id", middleware.RequireRole("creator", "admin"), a.DeleteModel)
	g.GET("/models", a.ListModels)
	g.GET("/models/:id", a.GetModel)
	g.POST("/models/:id/test", middleware.RequireRole("creator", "admin"), a.TestModel)
	g.PATCH("/models/:id/default", middleware.RequireRole("admin"), a.SetDefaultModel)

	// datasources
	g.POST("/datasources", middleware.RequireRole("creator", "admin"), a.CreateDatasource)
	g.PUT("/datasources/:id", middleware.RequireRole("creator", "admin"), a.UpdateDatasource)
	g.POST("/datasources/:id/clone", middleware.RequireRole("creator", "admin"), a.CloneDatasource)
	g.DELETE("/datasources/:id", middleware.RequireRole("creator", "admin"), a.DeleteDatasource)
	g.PATCH("/datasources/:id/status", middleware.RequireRole("creator", "admin"), a.PatchDatasourceStatus)
	g.POST("/datasources/:id/test", middleware.RequireRole("creator", "admin"), a.TestDatasource)
	g.GET("/datasources", a.ListDatasources)
	g.GET("/datasources/:id", a.GetDatasource)

	// files
	g.POST("/files", middleware.RequireRole("creator", "admin"), a.UploadFile)

	// agents
	g.POST("/agents", middleware.RequireRole("creator", "admin"), a.CreateAgent)
	g.PUT("/agents/:id", middleware.RequireRole("creator", "admin"), a.UpdateAgent)
	g.POST("/agents/:id/clone", middleware.RequireRole("creator", "admin"), a.CloneAgent)
	g.DELETE("/agents/:id", middleware.RequireRole("creator", "admin"), a.DeleteAgent)
	g.GET("/agents", a.ListAgents)
	g.GET("/agents/:id", a.GetAgent)
	g.POST("/agents/:id/preview", middleware.RequireRole("creator", "admin"), a.PreviewAgent)

	// workflows
	g.POST("/workflows", middleware.RequireRole("creator", "admin"), a.CreateWorkflow)
	g.PUT("/workflows/:id", middleware.RequireRole("creator", "admin"), a.UpdateWorkflow)
	g.POST("/workflows/:id/clone", a.CloneWorkflow)
	g.DELETE("/workflows/:id", middleware.RequireRole("creator", "admin"), a.DeleteWorkflow)
	g.GET("/workflows", a.ListWorkflows)
	g.GET("/workflows/:id", a.GetWorkflow)
	g.GET("/workflows/:id/versions", a.ListWorkflowVersions)
	g.POST("/workflows/:id/versions/:ver/rollback", middleware.RequireRole("creator", "admin"), a.RollbackWorkflow)
	g.GET("/workflows/:id/export", a.ExportWorkflow)
	g.POST("/workflows/import", middleware.RequireRole("creator", "admin"), a.ImportWorkflow)
	g.POST("/workflows/import/confirm", middleware.RequireRole("creator", "admin"), a.ConfirmImport)
	g.POST("/workflows/:id/share", middleware.RequireRole("creator", "admin"), a.ShareWorkflow)
	g.POST("/workflows/clone-by-code", a.CloneWorkflowByCode)
	g.PATCH("/workflows/:id/visibility", middleware.RequireRole("creator", "admin"), a.PatchWorkflowVisibility)

	// tasks
	g.POST("/tasks", a.CreateTask)
	g.GET("/tasks/:id", a.GetTask)
	g.GET("/tasks/:id/stream", a.TaskStream)
	g.POST("/tasks/:id/stop", a.StopTask)
	g.POST("/tasks/:id/rerun", a.RerunTask)
	g.GET("/tasks", a.ListTasks)
	g.GET("/tasks/:id/steps", a.ListTaskSteps)

	// reports
	g.GET("/reports", a.ListReports)
	g.GET("/reports/:id", a.GetReport)
	g.GET("/reports/:id/export/md", a.ExportReportMD)
	g.GET("/reports/:id/export/pdf", a.ExportReportPDF)
	g.GET("/reports/:id/export/docx", a.ExportReportDOCX)
	g.DELETE("/reports/:id", a.DeleteReport)
	g.PATCH("/reports/:id/archive", a.ArchiveReport)
	g.POST("/reports/batch-delete", a.BatchDeleteReports)

	// system
	sg := g.Group("/system")
	sg.Use(middleware.RequireRole("admin"))
	{
		sg.GET("/config", a.GetSystemConfig)
		sg.PATCH("/config", a.PatchSystemConfig)
		sg.GET("/dashboard", a.Dashboard)
		sg.GET("/audit-logs", a.ListAuditLogs)
	}
}

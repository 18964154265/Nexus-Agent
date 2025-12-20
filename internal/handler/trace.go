package handler

import (
	"context"
	"net/http"
	"time"

	"example.com/agent-server/internal/middleware"
	"example.com/agent-server/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// ListRuns 获取当前用户的所有任务列表
func (h *Handler) ListRuns(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	runs := h.Store.ListRunsByUser(userID)
	response.Success(ctx, runs)
}

// GetRunDetail 获取单个任务详情
func (h *Handler) GetRunDetail(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	runID := ctx.Param("id")
	run := h.Store.GetRun(runID)
	if run == nil || run.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Run not found")
		return
	}

	response.Success(ctx, run)
}

// RunStepResp 简单的 Trace 响应结构
type RunStepResp struct {
	ID            string                 `json:"id"`
	RunID         string                 `json:"run_id"`
	StepType      string                 `json:"step_type"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	InputPayload  map[string]interface{} `json:"input_payload"`
	OutputPayload map[string]interface{} `json:"output_payload"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	LatencyMS     int                    `json:"latency_ms"`
	StartedAt     string                 `json:"started_at"`
	FinishedAt    string                 `json:"finished_at"`
}

// GetRunTrace 获取任务的执行链路 (Trace)
func (h *Handler) GetRunTrace(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	runID := ctx.Param("id")
	run := h.Store.GetRun(runID)
	if run == nil || run.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Run not found")
		return
	}

	steps := h.Store.ListRunStepsByRun(runID)
	res := make([]*RunStepResp, 0, len(steps))
	for _, s := range steps {
		res = append(res, &RunStepResp{
			ID:            s.ID,
			RunID:         s.RunID,
			StepType:      s.StepType,
			Name:          s.Name,
			Status:        s.Status,
			InputPayload:  s.InputPayload,
			OutputPayload: s.OutputPayload,
			ErrorMessage:  s.ErrorMessage,
			LatencyMS:     s.LatencyMS,
			StartedAt:     s.StartedAt.Format(time.RFC3339),
			FinishedAt:    s.FinishedAt.Format(time.RFC3339),
		})
	}

	response.Success(ctx, res)
}

// CancelRun 异步取消任务
func (h *Handler) CancelRun(c context.Context, ctx *app.RequestContext) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.Unauthorized(ctx, "Unauthorized")
		return
	}

	runID := ctx.Param("id")
	run := h.Store.GetRun(runID)
	if run == nil || run.UserID != userID {
		response.Error(ctx, http.StatusNotFound, 40400, "Run not found")
		return
	}

	// 只有 running 状态的任务才需要取消
	// 但是为了幂等性，即使完成了也可以调 CancelRun，只是没效果
	if run.Status != "running" {
		response.Success(ctx, map[string]string{"message": "Run is not running"})
		return
	}

	// 调用 Engine 的取消方法
	if err := h.Engine.CancelRun(runID); err != nil {
		// 可能是已经结束了，或者 map 里没了，直接返回成功即可
	}

	response.Success(ctx, map[string]string{"message": "Cancellation requested"})
}

package handler

import (
	"context"
	"fmt"
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

// RunStepResp Trace 响应结构（支持树形结构）
type RunStepResp struct {
	ID            string                 `json:"id"`
	RunID         string                 `json:"run_id"`
	StepType      string                 `json:"step_type"` // 'thought', 'tool', 'run'
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	InputPayload  map[string]interface{} `json:"input_payload"`
	OutputPayload map[string]interface{} `json:"output_payload"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	LatencyMS     int                    `json:"latency_ms"`
	StartedAt     string                 `json:"started_at"`
	FinishedAt    string                 `json:"finished_at"`
	Children      []*RunStepResp         `json:"children,omitempty"` // 子步骤或子 Run
}

// GetRunTrace 获取任务的执行链路 (Trace) - 增强版，包含子 Run
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

	// 构建完整的 trace 树
	traceTree := h.buildTraceTree(runID, userID)
	response.Success(ctx, traceTree)
}

// buildTraceTree 构建 trace 树形结构
func (h *Handler) buildTraceTree(runID string, userID string) []*RunStepResp {
	// 1. 获取当前 Run 的所有 Steps
	steps := h.Store.ListRunStepsByRun(runID)

	// 2. 转换 Steps 为 RunStepResp
	stepResps := make([]*RunStepResp, 0, len(steps))
	for _, s := range steps {
		stepResp := &RunStepResp{
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
		}

		// 3. 检查这个 step 是否调用了子 Agent（通过检查 output 中是否有 child_run_id）
		// 或者通过工具调用的结果来判断
		if childRunID := extractChildRunID(s.OutputPayload); childRunID != "" {
			childRun := h.Store.GetRun(childRunID)
			if childRun != nil && childRun.UserID == userID {
				// 递归获取子 Run 的 trace
				childTrace := h.buildTraceTree(childRunID, userID)
				if len(childTrace) > 0 {
					// 创建一个 "run" 类型的节点来表示子 Agent 执行
					childRunResp := &RunStepResp{
						ID:            childRun.ID,
						RunID:         childRun.ID,
						StepType:      "run",
						Name:          fmt.Sprintf("Child Agent Run: %s", childRun.AgentID[:8]),
						Status:        childRun.Status,
						InputPayload:  childRun.InputPayload,
						OutputPayload: childRun.OutputPayload,
						LatencyMS:     calculateLatency(childRun.StartedAt, childRun.FinishedAt),
						StartedAt:     childRun.StartedAt.Format(time.RFC3339),
						FinishedAt:    childRun.FinishedAt.Format(time.RFC3339),
						Children:      childTrace,
					}
					stepResp.Children = []*RunStepResp{childRunResp}
				}
			}
		}

		stepResps = append(stepResps, stepResp)
	}

	// 4. 同时检查是否有直接通过 ParentRunID 关联的子 Run
	allRuns := h.Store.ListRunsByUser(userID)
	for _, r := range allRuns {
		if r.ParentRunID == runID {
			// 这是一个子 Run，创建 run 类型的节点
			childTrace := h.buildTraceTree(r.ID, userID)
			childRunResp := &RunStepResp{
				ID:            r.ID,
				RunID:         r.ID,
				StepType:      "run",
				Name:          fmt.Sprintf("Child Agent: %s", r.AgentID[:8]),
				Status:        r.Status,
				InputPayload:  r.InputPayload,
				OutputPayload: r.OutputPayload,
				LatencyMS:     calculateLatency(r.StartedAt, r.FinishedAt),
				StartedAt:     r.StartedAt.Format(time.RFC3339),
				FinishedAt:    r.FinishedAt.Format(time.RFC3339),
				Children:      childTrace,
			}
			stepResps = append(stepResps, childRunResp)
		}
	}

	return stepResps
}

// extractChildRunID 从 output payload 中提取子 Run ID
func extractChildRunID(output map[string]interface{}) string {
	if output == nil {
		return ""
	}

	// 尝试多种可能的字段名
	if childID, ok := output["child_run_id"].(string); ok && childID != "" {
		return childID
	}
	if childID, ok := output["run_id"].(string); ok && childID != "" {
		return childID
	}
	if result, ok := output["result"].(map[string]interface{}); ok {
		if childID, ok := result["run_id"].(string); ok && childID != "" {
			return childID
		}
	}

	return ""
}

// calculateLatency 计算延迟（毫秒）
func calculateLatency(started, finished time.Time) int {
	if finished.IsZero() {
		return 0
	}
	return int(finished.Sub(started).Milliseconds())
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

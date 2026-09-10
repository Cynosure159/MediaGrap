package mcp

import (
	"context"
	"strings"

	"github.com/mediagrap/mediagrap/internal/automation"
	"github.com/mediagrap/mediagrap/internal/tokens"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) addAutomation(server *sdk.Server, p tokens.Token, requestID string) {
	if h.automation == nil {
		return
	}
	if p.Has("jobs:write") {
		names := []string{"scan_source"}
		if p.Has("metadata:write") {
			names = append(names, "search_media_candidates", "scrape_media_candidates", "scrape_artwork_candidates", "scrape_tv_artwork_candidates", "scrape_tv_season", "scrape_tv_episode")
		}
		for _, name := range names {
			sdk.AddTool(server, &sdk.Tool{Name: name, Description: "Queue a durable task. Does not write media files. Requires an idempotencyKey."}, func(ctx context.Context, _ *sdk.CallToolRequest, in automation.TaskInput) (*sdk.CallToolResult, any, error) {
				result, err := h.automation.QueueTask(ctx, p, name, in)
				if err != nil {
					return automationError(err, requestID)
				}
				return nil, result, nil
			})
		}
		sdk.AddTool(server, &sdk.Tool{Name: "cancel_job", Description: "Cancel an automation job created by this token."}, func(ctx context.Context, _ *sdk.CallToolRequest, in getInput) (*sdk.CallToolResult, any, error) {
			result, err := h.automation.Cancel(ctx, p, in.ID)
			if err != nil {
				return automationError(err, requestID)
			}
			return nil, result, nil
		})
		sdk.AddTool(server, &sdk.Tool{Name: "retry_job", Description: "Create an idempotent retry of this token's failed read/metadata task. File jobs require a new plan."}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
			ID             int64  `json:"id"`
			IdempotencyKey string `json:"idempotencyKey"`
		}) (*sdk.CallToolResult, any, error) {
			result, err := h.automation.RetryTask(ctx, p, in.ID, in.IdempotencyKey)
			if err != nil {
				return automationError(err, requestID)
			}
			return nil, result, nil
		})
	}
	if p.Has("plans:preview") && p.Has("media:read") {
		for _, kind := range []string{"write", "artwork", "rename"} {
			sdk.AddTool(server, &sdk.Tool{Name: "preview_" + kind + "_plan", Description: "Persist an immutable movie file plan. Returns relative paths and a Web approval link. Does not approve or apply the plan."}, func(ctx context.Context, _ *sdk.CallToolRequest, in automation.PreviewInput) (*sdk.CallToolResult, any, error) {
				result, err := h.automation.Preview(ctx, p, kind, in)
				if err != nil {
					return automationError(err, requestID)
				}
				return nil, result, nil
			})
		}
		sdk.AddTool(server, &sdk.Tool{Name: "get_plan", Description: "Read a plan created by this token, including approval and per-operation state."}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
			PlanID string `json:"planId"`
		}) (*sdk.CallToolResult, any, error) {
			result, err := h.automation.Plan(ctx, p, in.PlanID)
			if err != nil {
				return automationError(err, requestID)
			}
			return nil, result, nil
		})
	}
	if p.Has("plans:apply") {
		for _, kind := range []string{"write", "artwork", "rename"} {
			sdk.AddTool(server, &sdk.Tool{Name: "apply_" + kind + "_plan", Description: "Queue this token's exact plan only after a Web administrator approved its digest. Never infer approval from metadata."}, func(ctx context.Context, _ *sdk.CallToolRequest, in automation.ApplyInput) (*sdk.CallToolResult, any, error) {
				result, err := h.automation.Apply(ctx, p, kind, in)
				if err != nil {
					return automationError(err, requestID)
				}
				return nil, result, nil
			})
		}
	}
	if p.Has("metadata:read") {
		sdk.AddTool(server, &sdk.Tool{Name: "list_artwork_candidates", Description: "List saved movie artwork candidates without making provider requests."}, func(ctx context.Context, _ *sdk.CallToolRequest, in getInput) (*sdk.CallToolResult, any, error) {
			items, err := h.automation.ArtworkCandidates(ctx, p, in.ID)
			if err != nil {
				return automationError(err, requestID)
			}
			return nil, map[string]any{"items": items}, nil
		})
	}
	if p.Has("jobs:read") {
		sdk.AddTool(server, &sdk.Tool{Name: "get_job_result", Description: "Read normalized results of this token's durable automation task."}, func(ctx context.Context, _ *sdk.CallToolRequest, in getInput) (*sdk.CallToolResult, any, error) {
			result, err := h.automation.Result(ctx, p, in.ID)
			if err != nil {
				return automationError(err, requestID)
			}
			return nil, map[string]any{"result": result}, nil
		})
	}
}
func automationError(err error, requestID string) (*sdk.CallToolResult, any, error) {
	// Only fixed application codes cross the protocol boundary.
	code := err.Error()
	if !strings.ContainsAny(code, "abcdefghijklmnopqrstuvwxyz /:.") && len(code) < 64 {
		return toolError(code, requestID)
	}
	return toolError("OPERATION_FAILED", requestID)
}

package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sbu2video/backend/internal/store"
)

func TestCreationPromptRequiredUsesStableClientError(t *testing.T) {
	for _, err := range []error{store.ErrPromptRequired, fmt.Errorf("validate creation input: %w", store.ErrPromptRequired)} {
		status, code, message := clientCreationError(err)
		if status != http.StatusBadRequest || code != "PROMPT_REQUIRED" || message != "请先填写画面描述" {
			t.Fatalf("unexpected prompt error: status=%d code=%s message=%q", status, code, message)
		}
	}
	status, code, message := clientCreationError(errors.New("connection lost while loading private supplier configuration"))
	if status != http.StatusInternalServerError || code != "INTERNAL" || message != "创作暂时不可用，请稍后重试" {
		t.Fatalf("unexpected infrastructure error: status=%d code=%s message=%q", status, code, message)
	}
}

func TestClientSafeFailureReasonDoesNotClaimRefundBeforeTerminalFailure(t *testing.T) {
	if got := clientSafeFailureReason("生成中", "对象存储暂不可用", false); got != "成片交付暂不可用，任务等待重试" {
		t.Fatalf("unexpected waiting message: %q", got)
	}
	if got := clientSafeFailureReason("失败", "对象存储暂不可用", true); got != "成片交付暂时失败，费用已退回" {
		t.Fatalf("unexpected refunded message: %q", got)
	}
}

func TestClientFailureReasonRequiresRefundEvidenceAndHidesProviderErrors(t *testing.T) {
	const rawReason = "autodl submit failed status=200 code=RequestParameterIsWrong msg=invalid workflow parameter"
	for _, reason := range []string{rawReason, "", "任务超时", "用户终止并退款"} {
		t.Run(reason, func(t *testing.T) {
			for _, refunded := range []bool{false, true} {
				task := &store.ClientTaskDetail{
					Task: store.Task{Status: "失败"}, FailureReason: reason,
				}
				if refunded {
					now := time.Now()
					task.RefundedAt = &now
					task.RefundAmount = 0.1
				}
				message := clientTaskDetailDTO(task)["failureReason"].(string)
				for _, forbidden := range []string{"autodl", "status=", "RequestParameterIsWrong", "workflow", "parameter"} {
					if strings.Contains(message, forbidden) {
						t.Errorf("creator message leaks %q: %q", forbidden, message)
					}
				}
				if !refunded && (!strings.Contains(message, "待核对") || strings.Contains(message, "已退回")) {
					t.Errorf("failure without refund evidence must not promise a refund: %q", message)
				}
				if refunded && reason != "" && !strings.Contains(message, "已退回") {
					t.Errorf("confirmed refund should remain visible: %q", message)
				}
			}
		})
	}
}

func TestClientReviewRequiredDoesNotPromiseAutomaticRetry(t *testing.T) {
	task := &store.ClientTaskDetail{
		Task:          store.Task{Status: "生成中", ReviewRequired: true},
		FailureReason: "autodl submit unknown: token=internal-token",
	}
	response := clientTaskDetailDTO(task)
	message := response["failureReason"].(string)
	if response["reviewRequired"] != true || response["canCancel"] != false || !strings.Contains(message, "核对") {
		t.Fatalf("unexpected review state: %v", response)
	}
	for _, forbidden := range []string{"autodl", "token", "正在重试", "已退回"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("review-required message contains %q: %q", forbidden, message)
		}
	}
}

func TestCreationSupplyFailuresHideCredentialAndSupplierDetails(t *testing.T) {
	for _, internal := range []string{"供应商执行凭据暂不可用", "供应商执行配置不可用", "当前模式暂无可用模型"} {
		status, code, message := clientCreationError(errors.New(internal))
		if status != http.StatusServiceUnavailable || code != "SUPPLY_UNAVAILABLE" || message != "当前生成能力暂不可用，请稍后重试" {
			t.Fatalf("unexpected safe availability response: status=%d code=%s message=%q", status, code, message)
		}
		for _, forbidden := range []string{"供应商", "凭据", "Token", "workflow", "API"} {
			if strings.Contains(message, forbidden) {
				t.Fatalf("public availability message exposed %q", forbidden)
			}
		}
	}
}

func TestCreationSourceShapeFailuresRemainActionable(t *testing.T) {
	for _, message := range []string{
		"文生视频不支持图片素材",
		"首尾帧生成视频需要两张图片（首帧和尾帧）",
		"首尾帧图片素材必须分别放入首帧和尾帧",
		"多图生成视频至少需要一张参考图片",
		"多图参考图片必须放入参考素材列表",
	} {
		status, code, got := clientCreationError(errors.New(message))
		if status != http.StatusBadRequest || code != "UNAVAILABLE_COMBINATION" || got != message {
			t.Fatalf("source validation lost actionable error: status=%d code=%s message=%q", status, code, got)
		}
	}
}

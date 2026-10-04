package store

import "testing"

func TestReconciliationRequiresExplicitEvidenceAndSafeTaskID(t *testing.T) {
	for _, in := range []TaskReconciliationInput{
		{Action: "retry", Confirmed: true, Evidence: "供应商已确认没有创建该任务"},
		{Action: "refund", Confirmed: false, Evidence: "供应商已确认没有创建该任务"},
		{Action: "refund", Confirmed: true, Evidence: "无"},
		{Action: "refund", Confirmed: true, Evidence: "供应商已确认没有创建该任务", UpstreamTaskID: "accepted"},
		{Action: "resume", Confirmed: true, Evidence: "已核对相同用户输入和提交时间", UpstreamTaskID: "../other"},
		{Action: "resume", Confirmed: true, Evidence: "已核对相同用户输入和提交时间"},
	} {
		if err := in.Validate(); err == nil {
			t.Fatalf("accepted invalid input %+v", in)
		}
	}
	for _, in := range []TaskReconciliationInput{
		{Action: "refund", Confirmed: true, Evidence: "供应商已确认没有创建该任务"},
		{Action: "resume", Confirmed: true, Evidence: "已核对相同用户输入和提交时间", UpstreamTaskID: "task-123_456"},
	} {
		if err := in.Validate(); err != nil {
			t.Fatalf("rejected valid input: %v", err)
		}
	}
}

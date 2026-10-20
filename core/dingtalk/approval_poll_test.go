package dingtalk

import (
	"errors"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/model"
)

type approvalPollFixture struct {
	calls    int
	response *dt.ProcessInstanceRespV2
	err      error
}

func (f *approvalPollFixture) CreateProcessInstance(*dt.CreateProcessInstanceInput, string) (string, error) {
	panic("unexpected create")
}
func (f *approvalPollFixture) GetProcessInstance(string, string) (*dt.ProcessInstanceRespV2, error) {
	f.calls++
	return f.response, f.err
}

func TestApprovalPollUnavailableResultsDoNotPanicOrChangePolicy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *dt.ProcessInstanceRespV2
		err      error
		calls    int
	}{
		{name: "qps limit", err: errors.New("403 Forbidden.AccessDenied.QpsLimitForApi"), calls: 1},
		{name: "HTTP 429", err: errors.New("response status code: 429"), calls: 1},
		{name: "upstream error", err: errors.New("upstream unavailable"), calls: 2},
		{name: "nil response", calls: 2},
		{name: "unsuccessful", response: &dt.ProcessInstanceRespV2{}, calls: 2},
		{name: "missing fields", response: &dt.ProcessInstanceRespV2{Success: true}, calls: 2},
		{name: "missing business ID", response: &dt.ProcessInstanceRespV2{Success: true, Result: dt.ProcessInstanceResult{Status: tea.String("COMPLETED"), Result: tea.String("agree")}}, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := newDingtalkUserTestApp(t)
			if err := service.DB.AutoMigrate(&model.Policy{}); err != nil {
				t.Fatal(err)
			}
			policies := []model.Policy{{ID: "one", Name: "one", ApprovalID: "pending-one", Users: model.ArrayString{"fixture"}, Actions: model.ArrayString{"connect"}, ExpiresAt: time.Now().Add(time.Hour)}, {ID: "two", Name: "two", ApprovalID: "pending-two", Users: model.ArrayString{"fixture"}, Actions: model.ArrayString{"connect"}, ExpiresAt: time.Now().Add(time.Hour)}}
			for _, p := range policies {
				if err := service.DB.Create(&p).Error; err != nil {
					t.Fatal(err)
				}
			}
			f := &approvalPollFixture{response: tc.response, err: tc.err}
			if got := syncApprovals(policies, f, "fixture-token"); len(got) != 0 {
				t.Fatal("unavailable result accepted")
			}
			if f.calls != tc.calls {
				t.Fatalf("calls=%d want=%d", f.calls, tc.calls)
			}
			var unchanged []model.Policy
			if err := service.DB.Find(&unchanged).Error; err != nil {
				t.Fatal(err)
			}
			for _, p := range unchanged {
				if p.IsEnabled || p.Approver != "" {
					t.Fatal("failed lookup changed authorization")
				}
			}
		})
	}
}

func TestApprovalPollSuccessAndTerminationWithoutOptionalFields(t *testing.T) {
	for _, status := range []string{"COMPLETED", "TERMINATED"} {
		t.Run(status, func(t *testing.T) {
			service := newDingtalkUserTestApp(t)
			if err := service.DB.AutoMigrate(&model.Policy{}); err != nil {
				t.Fatal(err)
			}
			policy := model.Policy{ID: "fixture", Name: "fixture", ApprovalID: "pending", Users: model.ArrayString{"fixture"}, Actions: model.ArrayString{"connect"}, ExpiresAt: time.Now().Add(time.Hour)}
			if err := service.DB.Create(&policy).Error; err != nil {
				t.Fatal(err)
			}
			result := dt.ProcessInstanceResult{Status: tea.String(status)}
			if status == "COMPLETED" {
				result.Result = tea.String("agree")
				result.BusinessId = tea.String("fixture-business")
			}
			f := &approvalPollFixture{response: &dt.ProcessInstanceRespV2{Success: true, Result: result}}
			if got := syncApprovals([]model.Policy{policy}, f, "fixture-token"); len(got) != 1 {
				t.Fatal("valid result not applied")
			}
			var got model.Policy
			if err := service.DB.First(&got, "id = ?", policy.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.IsEnabled != (status == "COMPLETED") || got.Approver == "" {
				t.Fatal("incorrect approval state")
			}
		})
	}
}

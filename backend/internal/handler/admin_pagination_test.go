package handler

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminPageQueryDefaultsAndBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name               string
		url                string
		wantPage, wantSize int
		wantPaged          bool
	}{
		{name: "legacy request remains unpaged", url: "/admin/suppliers", wantPaged: false},
		{name: "defaults missing values", url: "/admin/suppliers?page=0&pageSize=bad", wantPage: 1, wantSize: 20, wantPaged: true},
		{name: "caps oversized page size", url: "/admin/suppliers?page=3&pageSize=1000", wantPage: 3, wantSize: 100, wantPaged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = req
			page, pageSize, paged := adminPageQuery(ctx)
			if page != tc.wantPage || pageSize != tc.wantSize || paged != tc.wantPaged {
				t.Fatalf("adminPageQuery() = (%d, %d, %t), want (%d, %d, %t)", page, pageSize, paged, tc.wantPage, tc.wantSize, tc.wantPaged)
			}
		})
	}
}

func TestAdminCSVQuery(t *testing.T) {
	if got, want := adminCSVQuery(" exec_a, ,exec_b,,"), []string{"exec_a", "exec_b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("adminCSVQuery() = %#v, want %#v", got, want)
	}
	if got := adminCSVQuery(" "); got != nil {
		t.Fatalf("adminCSVQuery(blank) = %#v, want nil", got)
	}
}

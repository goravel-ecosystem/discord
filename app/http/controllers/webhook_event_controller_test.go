package controllers

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"testing"

	"github.com/goravel/framework/contracts/database/db"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

func TestWebhookEventControllerIndexDefaults(t *testing.T) {
	env := newControllerEnv(t)
	env.expectIndex(1, 50, "", "")
	controller := &WebhookEventController{}

	controller.Index(env.httpContext)

	require.Equal(t, 1, env.modelCalls)
	require.Equal(t, []string{"id"}, env.orderByDescCalls)
	require.Equal(t, 1, env.paginateCalls)
	require.Equal(t, 1, env.paginatePage)
	require.Equal(t, 50, env.paginateLimit)
	require.Empty(t, env.whereQueries)

	require.Equal(t, 1, env.jsonCalls)
	require.Equal(t, stdhttp.StatusOK, env.jsonCode)
	body := env.jsonObj.(contractshttp.Json)
	require.Equal(t, 1, body["page"])
	require.Equal(t, 50, body["limit"])
	require.Equal(t, int64(0), body["total"])

	// Empty result serialises as [] rather than null.
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"data":[]`)
}

func TestWebhookEventControllerIndexPagingBounds(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		wantPage  int
		wantLimit int
	}{
		{name: "page zero clamps to one", page: 0, limit: 50, wantPage: 1, wantLimit: 50},
		{name: "page above max clamps", page: 100001, limit: 50, wantPage: 100000, wantLimit: 50},
		{name: "limit zero clamps to fifty", page: 1, limit: 0, wantPage: 1, wantLimit: 50},
		{name: "limit above max clamps", page: 1, limit: 501, wantPage: 1, wantLimit: 500},
		{name: "limit negative clamps to fifty", page: 1, limit: -1, wantPage: 1, wantLimit: 50},
		{name: "explicit values pass through", page: 3, limit: 10, wantPage: 3, wantLimit: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newControllerEnv(t)
			env.expectIndex(test.page, test.limit, "", "")
			controller := &WebhookEventController{}

			controller.Index(env.httpContext)

			require.Equal(t, 1, env.paginateCalls)
			require.Equal(t, test.wantPage, env.paginatePage)
			require.Equal(t, test.wantLimit, env.paginateLimit)

			body := env.jsonObj.(contractshttp.Json)
			require.Equal(t, test.wantPage, body["page"])
			require.Equal(t, test.wantLimit, body["limit"])
		})
	}
}

func TestWebhookEventControllerIndexFilters(t *testing.T) {
	t.Run("platform and event filters reach Where", func(t *testing.T) {
		env := newControllerEnv(t)
		env.expectIndex(1, 50, "github", "push")
		controller := &WebhookEventController{}

		controller.Index(env.httpContext)

		require.Equal(t, []any{"platform", "event"}, env.whereQueries)
		require.Len(t, env.whereArgs, 2)
		require.Equal(t, []any{"github"}, env.whereArgs[0])
		require.Equal(t, []any{"push"}, env.whereArgs[1])

		require.Equal(t, []string{"platform", "event"}, env.queryCalls)
	})

	t.Run("platform filter only reaches Where once", func(t *testing.T) {
		env := newControllerEnv(t)
		env.expectIndex(1, 50, "github", "")
		controller := &WebhookEventController{}

		controller.Index(env.httpContext)

		require.Equal(t, []any{"platform"}, env.whereQueries)
		require.Len(t, env.whereArgs, 1)
		require.Equal(t, []any{"github"}, env.whereArgs[0])
		require.Equal(t, []string{"platform", "event"}, env.queryCalls)
	})

	t.Run("event filter only reaches Where once", func(t *testing.T) {
		env := newControllerEnv(t)
		env.expectIndex(1, 50, "", "push")
		controller := &WebhookEventController{}

		controller.Index(env.httpContext)

		require.Equal(t, []any{"event"}, env.whereQueries)
		require.Len(t, env.whereArgs, 1)
		require.Equal(t, []any{"push"}, env.whereArgs[0])
		require.Equal(t, []string{"platform", "event"}, env.queryCalls)
	})

	t.Run("absent filters do not add Where clauses", func(t *testing.T) {
		env := newControllerEnv(t)
		env.expectIndex(1, 50, "", "")
		controller := &WebhookEventController{}

		controller.Index(env.httpContext)

		require.Empty(t, env.whereQueries)
	})
}

func TestWebhookEventControllerIndexPopulatesRows(t *testing.T) {
	env := newControllerEnv(t)
	env.expectIndex(1, 50, "", "")
	env.paginateFn = func(page, limit int, dest any, total *int64) error {
		events := dest.(*[]models.WebhookEvent)
		*events = append(*events, models.WebhookEvent{Platform: models.PlatformGithub})
		*total = 7

		return nil
	}
	controller := &WebhookEventController{}

	controller.Index(env.httpContext)

	body := env.jsonObj.(contractshttp.Json)
	require.Equal(t, int64(7), body["total"])
	data := body["data"].([]models.WebhookEvent)
	require.Len(t, data, 1)
	require.Equal(t, models.PlatformGithub, data[0].Platform)
}

func TestWebhookEventControllerIndexPaginateError(t *testing.T) {
	env := newControllerEnv(t)
	env.expectIndex(1, 50, "", "")
	env.paginateErr = errors.New("db down")
	controller := &WebhookEventController{}

	controller.Index(env.httpContext)

	require.Equal(t, 1, env.jsonCalls)
	require.Equal(t, stdhttp.StatusInternalServerError, env.jsonCode)
	require.Equal(t, contractshttp.Json{"error": "internal error"}, env.jsonObj)
	require.NotEmpty(t, env.logs)
}

func TestWebhookEventControllerDestroy(t *testing.T) {
	tests := []struct {
		name         string
		id           int64
		deleteResult *db.Result
		deleteErr    error
		wantCode     int
		wantQuery    bool
		wantLogged   bool
	}{
		{
			name:         "valid id deletes and returns 200",
			id:           1,
			deleteResult: &db.Result{RowsAffected: 1},
			wantCode:     stdhttp.StatusOK,
			wantQuery:    true,
		},
		{
			name:         "missing row returns 404",
			id:           2,
			deleteResult: &db.Result{RowsAffected: 0},
			wantCode:     stdhttp.StatusNotFound,
			wantQuery:    true,
		},
		{
			name:      "zero id returns 404 without a query",
			id:        0,
			wantCode:  stdhttp.StatusNotFound,
			wantQuery: false,
		},
		{
			name:      "negative id returns 404 without a query",
			id:        -5,
			wantCode:  stdhttp.StatusNotFound,
			wantQuery: false,
		},
		{
			name:       "delete error returns 500",
			id:         3,
			deleteErr:  errors.New("db down"),
			wantCode:   stdhttp.StatusInternalServerError,
			wantQuery:  true,
			wantLogged: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newControllerEnv(t)
			env.deleteResult = test.deleteResult
			env.deleteErr = test.deleteErr
			env.request.On("RouteInt64", "id").Return(test.id).Run(func(args mock.Arguments) {
				env.routeInt64Calls = append(env.routeInt64Calls, args.Get(0).(string))
			})
			controller := &WebhookEventController{}

			controller.Destroy(env.httpContext)

			require.Equal(t, []string{"id"}, env.routeInt64Calls)
			require.Len(t, env.noContentCodes, 1)
			require.Equal(t, test.wantCode, env.noContentCodes[0])

			if test.wantQuery {
				require.Equal(t, 1, env.ormQueryCalls)
				require.Len(t, env.deleteArgs, 1)
				require.IsType(t, &models.WebhookEvent{}, env.deleteArgs[0][0])
				require.Equal(t, []any{"id"}, env.whereQueries)
				require.Len(t, env.whereArgs, 1)
				require.Equal(t, []any{test.id}, env.whereArgs[0])
			} else {
				require.Zero(t, env.ormQueryCalls)
				require.Empty(t, env.deleteArgs)
			}

			if test.wantLogged {
				require.NotEmpty(t, env.logs)
			} else {
				require.Empty(t, env.logs)
			}
		})
	}
}

package controllers

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/goravel/framework/contracts/database/db"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/mail"
	"github.com/goravel/framework/foundation"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mockshttp "github.com/goravel/framework/mocks/http"
	mockslog "github.com/goravel/framework/mocks/log"
	mocksmail "github.com/goravel/framework/mocks/mail"
	frameworkmock "github.com/goravel/framework/testing/mock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"goravel/app/services"
)

// fakeGithub implements services.Github, an application-owned interface the
// framework mock factory does not cover, so it keeps a hand-written double.
type fakeGithub struct {
	services.Github

	err    error
	origin *stdhttp.Request
	calls  int
}

func (g *fakeGithub) ProcessWebhook(request *stdhttp.Request) error {
	g.calls++
	g.origin = request

	return g.err
}

// controllerEnv wires the framework mock factory (config, log, orm, mail and
// the http contracts) for the duration of a test, and captures the arguments
// the controller tests assert on. The mocks are locals in newControllerEnv; only
// the handles the tests read are kept as fields.
type controllerEnv struct {
	request     *mockshttp.ContextRequest
	httpContext *mockshttp.Context
	config      *mocksconfig.Config
	mail        *mocksmail.Mail

	// Captured from mock calls.
	logs             []string
	jsonCalls        int
	jsonCode         int
	jsonObj          any
	noContentCodes   []int
	modelCalls       int
	orderByDescCalls []string
	whereQueries     []any
	whereArgs        [][]any
	paginateCalls    int
	paginatePage     int
	paginateLimit    int
	deleteArgs       [][]any
	ormQueryCalls    int
	queryCalls       []string
	routeInt64Calls  []string
	mailTo           []string
	mailSubject      string
	mailContent      mail.Content
	mailSends        int

	// Scripted behaviour consulted by the mock query.
	paginateFn   func(page, limit int, dest any, total *int64) error
	paginateErr  error
	deleteResult *db.Result
	deleteErr    error
}

func newControllerEnv(t *testing.T) *controllerEnv {
	t.Helper()

	// mock.Factory() installs a mocked application as the package-level
	// foundation.App, so restore the previous one when the test finishes. This
	// avoids ever booting the real app (which would open a live Discord session).
	originalApp := foundation.App
	mockFactory := frameworkmock.Factory()
	t.Cleanup(func() { foundation.App = originalApp })

	appMock := mockFactory.App()
	appMock.Test(t)

	configMock := mockFactory.Config()
	configMock.Test(t)

	logMock := &mockslog.Log{}
	logMock.Test(t)
	appMock.On("MakeLog").Return(logMock)

	mailMock := mockFactory.Mail()
	mailMock.Test(t)

	ormMock := mockFactory.Orm()
	ormMock.Test(t)
	queryMock := mockFactory.OrmQuery()
	queryMock.Test(t)

	requestMock := mockFactory.ContextRequest()
	requestMock.Test(t)
	responseMock := mockFactory.ContextResponse()
	responseMock.Test(t)
	contextMock := mockFactory.Context()
	contextMock.Test(t)
	contextMock.On("Request").Return(requestMock)
	contextMock.On("Response").Return(responseMock)

	env := &controllerEnv{
		request:     requestMock,
		httpContext: contextMock,
		config:      configMock,
		mail:        mailMock,
	}

	// Assert the mocks whose expectations this env keeps required, once per test.
	// Only those can actually fail: the context/request mocks (driven by the
	// per-test helpers), mail (only registered by the notification subtest) and
	// the config recipient registered there. The query/log/response/orm mocks are
	// marked .Maybe() because each test exercises a different subset, so their
	// assertions are structural only. The factory-backed app mock is excluded:
	// its Make* expectations come from the factory's expecter and cannot be marked
	// .Maybe(), so asserting it would fail tests that legitimately skip a facade.
	t.Cleanup(func() {
		configMock.AssertExpectations(t)
		contextMock.AssertExpectations(t)
		logMock.AssertExpectations(t)
		mailMock.AssertExpectations(t)
		ormMock.AssertExpectations(t)
		queryMock.AssertExpectations(t)
		requestMock.AssertExpectations(t)
		responseMock.AssertExpectations(t)
	})

	ormMock.On("Query").Return(queryMock).Maybe().Run(func(mock.Arguments) {
		env.ormQueryCalls++
	})

	logMock.On("Error", mock.Anything).Maybe().Run(func(args mock.Arguments) {
		env.logs = append(env.logs, fmt.Sprint(args...))
	})
	responseMock.On("Json", mock.Anything, mock.Anything).Return(nil).Maybe().Run(func(args mock.Arguments) {
		env.jsonCalls++
		env.jsonCode = args.Get(0).(int)
		env.jsonObj = args.Get(1)
	})
	responseMock.On("NoContent", mock.Anything).Return(nil).Maybe().Run(func(args mock.Arguments) {
		env.noContentCodes = append(env.noContentCodes, args.Get(0).(int))
	})

	// The query mock returns itself from every chainable call so the production
	// builder chain keeps working while the calls are recorded.
	queryMock.On("Model", mock.Anything).Return(queryMock).Maybe().Run(func(mock.Arguments) {
		env.modelCalls++
	})
	queryMock.On("OrderByDesc", mock.Anything).Return(queryMock).Maybe().Run(func(args mock.Arguments) {
		env.orderByDescCalls = append(env.orderByDescCalls, args.Get(0).(string))
	})
	queryMock.On("Where", mock.Anything, mock.Anything).Return(queryMock).Maybe().Run(func(args mock.Arguments) {
		env.whereQueries = append(env.whereQueries, args.Get(0))
		env.whereArgs = append(env.whereArgs, append([]any(nil), args[1:]...))
	})
	queryMock.On("Paginate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(func(page, limit int, dest any, total *int64) error {
		env.paginateCalls++
		env.paginatePage = page
		env.paginateLimit = limit
		if env.paginateFn != nil {
			return env.paginateFn(page, limit, dest, total)
		}

		return env.paginateErr
	}).Maybe()
	queryMock.On("Delete", mock.Anything).Return(func(value ...any) (*db.Result, error) {
		env.deleteArgs = append(env.deleteArgs, append([]any(nil), value...))
		if env.deleteErr != nil {
			return nil, env.deleteErr
		}
		if env.deleteResult != nil {
			return env.deleteResult, nil
		}

		return &db.Result{RowsAffected: 1}, nil
	}).Maybe()

	return env
}

// expectIndex wires the query-string accessors an Index call performs, returning
// page/limit for the paging keys and platform/event for the filters.
func (e *controllerEnv) expectIndex(page, limit int, platform, event string) {
	e.request.On("QueryInt", "page", 1).Return(page)
	e.request.On("QueryInt", "limit", 50).Return(limit)
	e.request.On("Query", "platform").Return(platform).Run(func(args mock.Arguments) {
		e.queryCalls = append(e.queryCalls, args.Get(0).(string))
	})
	e.request.On("Query", "event").Return(event).Run(func(args mock.Arguments) {
		e.queryCalls = append(e.queryCalls, args.Get(0).(string))
	})
}

// expectMail wires the mail chain a failed webhook triggers. The recipient
// expectation is scenario-scoped here so the cleanup assertion can fail when the
// controller stops reading mail.to from config.
func (e *controllerEnv) expectMail() {
	e.config.On("GetString", "mail.to").Return("ops@example.com")
	e.mail.On("To", mock.Anything).Return(e.mail).Run(func(args mock.Arguments) {
		e.mailTo = args.Get(0).([]string)
	})
	e.mail.On("Subject", mock.Anything).Return(e.mail).Run(func(args mock.Arguments) {
		e.mailSubject = args.Get(0).(string)
	})
	e.mail.On("Content", mock.Anything).Return(e.mail).Run(func(args mock.Arguments) {
		e.mailContent = args.Get(0).(mail.Content)
	})
	e.mail.On("Send").Return(nil).Run(func(mock.Arguments) {
		e.mailSends++
	})
}

func TestWebhookControllerGithub(t *testing.T) {
	origin := httptest.NewRequest(stdhttp.MethodPost, "/webhooks/github", nil)

	tests := []struct {
		name       string
		processErr error
		wantCode   int
		wantBody   contractshttp.Json
		wantMail   bool
		wantLogged bool
	}{
		{
			name:       "success returns 200",
			processErr: nil,
			wantCode:   stdhttp.StatusOK,
			wantBody:   contractshttp.Json{"message": "Webhook processed successfully"},
		},
		{
			name:       "missing secret returns 401",
			processErr: services.ErrMissingSecret,
			wantCode:   stdhttp.StatusUnauthorized,
			wantBody:   contractshttp.Json{"error": "invalid signature"},
			wantLogged: true,
		},
		{
			name:       "invalid signature returns 401",
			processErr: fmt.Errorf("%w: %w", services.ErrInvalidSignature, errors.New("bad signature")),
			wantCode:   stdhttp.StatusUnauthorized,
			wantBody:   contractshttp.Json{"error": "invalid signature"},
			wantLogged: true,
		},
		{
			name:       "unexpected error returns 400 and notifies",
			processErr: errors.New("discord is down"),
			wantCode:   stdhttp.StatusBadRequest,
			wantBody:   contractshttp.Json{"error": "discord is down"},
			wantMail:   true,
			wantLogged: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newControllerEnv(t)
			env.request.On("Origin").Return(origin)
			if test.wantMail {
				env.expectMail()
			}
			github := &fakeGithub{err: test.processErr}
			controller := &WebhookController{github: github}

			controller.Github(env.httpContext)

			require.Equal(t, 1, github.calls)
			require.Same(t, origin, github.origin)

			require.Equal(t, 1, env.jsonCalls)
			require.Equal(t, test.wantCode, env.jsonCode)
			require.Equal(t, test.wantBody, env.jsonObj)

			if test.wantMail {
				require.Equal(t, 1, env.mailSends)
				require.Equal(t, []string{"ops@example.com"}, env.mailTo)
				require.Equal(t, "Github Webhook Failed", env.mailSubject)
				require.Equal(t, test.processErr.Error(), env.mailContent.Html)
			} else {
				require.Empty(t, env.mail.Calls)
			}

			if test.wantLogged {
				require.NotEmpty(t, env.logs)
			} else {
				require.Empty(t, env.logs)
			}
		})
	}
}

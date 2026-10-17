package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/goravel/framework/contracts/database/db"
	"github.com/goravel/framework/foundation"
	mockslog "github.com/goravel/framework/mocks/log"
	frameworkmock "github.com/goravel/framework/testing/mock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

// fakeDiscord implements the Discord interface and records every interaction.
// The framework mock factory only covers framework contracts, so this
// application-owned interface keeps a hand-written double.
type fakeDiscord struct {
	createThreadCalls []createThreadCall
	sendMessageCalls  []sendMessageCall
	deleteThreadCalls []string
	closeCalls        int

	createThreadID  string
	createThreadErr error
	sendMessageErr  error
	deleteThreadErr error
}

type createThreadCall struct {
	channelID string
	thread    Thread
}

type sendMessageCall struct {
	channelID string
	message   string
}

func (d *fakeDiscord) Close() { d.closeCalls++ }

func (d *fakeDiscord) CreateThread(channelID string, thread Thread) (string, error) {
	d.createThreadCalls = append(d.createThreadCalls, createThreadCall{channelID: channelID, thread: thread})
	if d.createThreadErr != nil {
		return "", d.createThreadErr
	}

	return d.createThreadID, nil
}

func (d *fakeDiscord) SendMessage(channelID string, message string) error {
	d.sendMessageCalls = append(d.sendMessageCalls, sendMessageCall{channelID: channelID, message: message})

	return d.sendMessageErr
}

func (d *fakeDiscord) DeleteThread(threadID string) error {
	d.deleteThreadCalls = append(d.deleteThreadCalls, threadID)

	return d.deleteThreadErr
}

const testSecret = "test-webhook-secret"

// testEnv wires the framework mock factory (config, orm, log) together with the
// hand-written Discord double for the duration of a test.
type testEnv struct {
	discord *fakeDiscord
	github  *GithubImpl
	secret  string

	// Scripted behaviour consulted by the mock query.
	firstFn         func(dest any) error
	firstOrCreateFn func(dest any, conds ...any) error
	deleteResult    *db.Result
	deleteErr       error

	// Arguments captured from the mock query.
	storedEvents     []*models.WebhookEvent
	lookupEvents     []*models.WebhookEvent
	pullRequestRows  []*models.PullRequest
	deleteCalls      [][]any
	withTrashedCalls int
	whereQueries     []any
	whereArgs        [][]any

	// Messages captured from the log mock.
	logs []string
}

func newTestEnv(t *testing.T) *testEnv {
	return newTestEnvWith(t, nil)
}

// newTestEnvWith builds the environment with the given config overrides applied
// on top of the defaults.
func newTestEnvWith(t *testing.T, configOverrides map[string]string) *testEnv {
	t.Helper()

	// mock.Factory() installs a mocked application as the package-level
	// foundation.App, so restore the previous one when the test finishes.
	originalApp := foundation.App
	mockFactory := frameworkmock.Factory()
	t.Cleanup(func() { foundation.App = originalApp })

	appMock := mockFactory.App()
	appMock.Test(t)

	configMock := mockFactory.Config()
	configMock.Test(t)
	values := map[string]string{
		"github.webhook_secret":            testSecret,
		"discord.roles.core":               "core-role",
		"discord.pull_requests.channel_id": "pr-channel",
	}
	for key, value := range configOverrides {
		values[key] = value
	}
	for key, value := range values {
		configMock.On("GetString", key).Return(value)
	}

	ormMock := mockFactory.Orm()
	ormMock.Test(t)
	queryMock := mockFactory.OrmQuery()
	queryMock.Test(t)
	// Only tests that reach the ORM call Query, so the expectation is optional.
	ormMock.On("Query").Return(queryMock).Maybe()

	logMock := &mockslog.Log{}
	logMock.Test(t)
	appMock.On("MakeLog").Return(logMock)

	// Assert the mocks whose expectations this env registers once per test.
	// newGithub always reads every config key, so an unread expectation there
	// means the config-to-field wiring regressed; orm/log register optional
	// expectations (not every scenario reaches the ORM or logs), so their
	// assertions are structural only.
	t.Cleanup(func() {
		configMock.AssertExpectations(t)
		ormMock.AssertExpectations(t)
		logMock.AssertExpectations(t)
	})

	discordFake := &fakeDiscord{createThreadID: "thread-created"}

	env := &testEnv{
		discord: discordFake,
		secret:  testSecret,
	}

	// The query mock returns itself from every chainable call so the production
	// builder chain keeps working while the calls are recorded.
	queryMock.On("WithTrashed").Return(queryMock).Maybe().Run(func(mock.Arguments) {
		env.withTrashedCalls++
	})
	queryMock.On("Where", mock.Anything, mock.Anything).Return(queryMock).Maybe().Run(func(args mock.Arguments) {
		env.whereQueries = append(env.whereQueries, args.Get(0))
		env.whereArgs = append(env.whereArgs, append([]any(nil), args[1:]...))
	})
	queryMock.On("First", mock.Anything).Return(func(dest any) error {
		if env.firstFn != nil {
			return env.firstFn(dest)
		}

		return nil
	}).Maybe()
	queryMock.On("FirstOrCreate", mock.Anything, mock.Anything).Return(func(dest any, conds ...any) error {
		if len(conds) > 0 {
			if row, ok := conds[0].(*models.PullRequest); ok {
				env.pullRequestRows = append(env.pullRequestRows, row)
			}
		}
		if env.firstOrCreateFn != nil {
			return env.firstOrCreateFn(dest, conds...)
		}

		return nil
	}).Maybe()
	queryMock.On("FirstOrCreate", mock.Anything, mock.Anything, mock.Anything).Return(func(dest any, conds ...any) error {
		if len(conds) > 0 {
			if lookup, ok := conds[0].(*models.WebhookEvent); ok {
				env.lookupEvents = append(env.lookupEvents, lookup)
			}
		}
		if len(conds) > 1 {
			if stored, ok := conds[1].(*models.WebhookEvent); ok {
				env.storedEvents = append(env.storedEvents, stored)
			}
		}
		if env.firstOrCreateFn != nil {
			return env.firstOrCreateFn(dest, conds...)
		}

		return nil
	}).Maybe()
	queryMock.On("Delete", mock.Anything).Return(func(value ...any) (*db.Result, error) {
		env.deleteCalls = append(env.deleteCalls, append([]any(nil), value...))
		if env.deleteErr != nil {
			return nil, env.deleteErr
		}
		if env.deleteResult != nil {
			return env.deleteResult, nil
		}

		return &db.Result{RowsAffected: 1}, nil
	}).Maybe()

	logMock.On("Error", mock.Anything).Maybe().Run(func(args mock.Arguments) {
		env.logs = append(env.logs, fmt.Sprint(args...))
	})

	// Build the subject through the real constructor so the config-to-field
	// wiring stays covered.
	impl, err := newGithub(discordFake)
	require.NoError(t, err)
	env.github = impl

	return env
}

// sign computes the X-Hub-Signature-256 value for a payload.
func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))

	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// signedRequest builds a POST webhook request with a valid signature.
func signedRequest(secret, event, delivery, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
	request.Header.Set("X-GitHub-Event", event)
	request.Header.Set("X-GitHub-Delivery", delivery)
	request.Header.Set("X-Hub-Signature-256", sign(secret, body))

	return request
}

// pullRequestBody builds a realistic github.PullRequestPayload JSON document.
func pullRequestBody(t *testing.T, action string, labels ...string) string {
	t.Helper()

	labelObjs := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		labelObjs = append(labelObjs, map[string]any{"name": label})
	}

	payload := map[string]any{
		"action": action,
		"number": 7,
		"pull_request": map[string]any{
			"id":       123,
			"number":   7,
			"title":    "Add tests",
			"html_url": "https://github.com/goravel/discord/pull/7",
			"state":    "open",
			"user":     map[string]any{"login": "bowen", "html_url": "https://github.com/bowen"},
			"labels":   labelObjs,
		},
		"repository": map[string]any{"full_name": "goravel/discord", "html_url": "https://github.com/goravel/discord"},
		"sender":     map[string]any{"login": "bowen"},
	}

	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	return string(raw)
}

// parsePullRequestPayload decodes a payload body into the library type.
func parsePullRequestPayload(t *testing.T, body string) github.PullRequestPayload {
	t.Helper()

	var payload github.PullRequestPayload
	require.NoError(t, json.Unmarshal([]byte(body), &payload))

	return payload
}

// populatePullRequestRow returns a fake First that reports an existing row so
// the pullRequest.ID == 0 guard in ProcessWebhook passes.
func populatePullRequestRow(t *testing.T) func(dest any) error {
	t.Helper()

	return func(dest any) error {
		pullRequest, ok := dest.(*models.PullRequest)
		require.True(t, ok, "dest = %#v, want *models.PullRequest", dest)
		pullRequest.ID = 5
		pullRequest.DiscordThreadID = "thread-existing"

		return nil
	}
}

func TestIsSignatureError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "hmac verification failed", err: github.ErrHMACVerificationFailed, want: true},
		{name: "missing signature header", err: github.ErrMissingHubSignatureHeader, want: true},
		{name: "missing event header", err: github.ErrMissingGithubEventHeader, want: true},
		{
			name: "wrapped hmac verification failed",
			err:  fmt.Errorf("%w: %w", ErrInvalidSignature, github.ErrHMACVerificationFailed),
			want: true,
		},
		{
			name: "wrapped missing signature header",
			err:  fmt.Errorf("%w: %w", ErrInvalidSignature, github.ErrMissingHubSignatureHeader),
			want: true,
		},
		{
			name: "wrapped missing event header",
			err:  fmt.Errorf("%w: %w", ErrInvalidSignature, github.ErrMissingGithubEventHeader),
			want: true,
		},
		{name: "unrelated error", err: errors.New("boom"), want: false},
		{name: "nil error", err: nil, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, isSignatureError(test.err))
		})
	}
}

func TestProcessWebhookMissingSecret(t *testing.T) {
	env := newTestEnvWith(t, map[string]string{"github.webhook_secret": ""})

	err := env.github.ProcessWebhook(signedRequest(env.secret, "push", "delivery-1", `{"action":"opened"}`))

	require.ErrorIs(t, err, ErrMissingSecret)
	require.Empty(t, env.storedEvents)
}

func TestProcessWebhookRejectsInvalidRequests(t *testing.T) {
	const body = `{"action":"opened","repository":{"full_name":"goravel/discord"},"sender":{"login":"bowen"}}`

	tests := []struct {
		name      string
		event     string
		delivery  string
		signature string
		body      string
	}{
		{name: "bad signature", event: "push", delivery: "d-bad", signature: "sha256=deadbeef", body: body},
		{name: "missing signature header", event: "push", delivery: "d-nosig", signature: "", body: body},
		{name: "missing event header", event: "", delivery: "d-noevent", signature: sign(testSecret, body), body: body},
		{name: "missing delivery header", event: "push", delivery: "", signature: sign(testSecret, body), body: body},
		{name: "empty body", event: "push", delivery: "d-empty", signature: sign(testSecret, ""), body: ""},
		{name: "oversized body", event: "push", delivery: "d-big", signature: sign(testSecret, ""), body: strings.Repeat("a", maxWebhookBodyBytes+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t)

			request := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(test.body))
			if test.event != "" {
				request.Header.Set("X-GitHub-Event", test.event)
			}
			if test.delivery != "" {
				request.Header.Set("X-GitHub-Delivery", test.delivery)
			}
			if test.signature != "" {
				request.Header.Set("X-Hub-Signature-256", test.signature)
			}

			err := env.github.ProcessWebhook(request)

			require.ErrorIs(t, err, ErrInvalidSignature)
			require.Empty(t, env.storedEvents)
		})
	}
}

func TestProcessWebhookValidSignatureStoresEvent(t *testing.T) {
	env := newTestEnv(t)
	body := `{"action":"opened","repository":{"full_name":"goravel/framework"},"sender":{"login":"bowen"}}`

	err := env.github.ProcessWebhook(signedRequest(env.secret, "push", "delivery-valid", body))

	require.NoError(t, err)
	require.Len(t, env.storedEvents, 1)

	stored := env.storedEvents[0]
	require.Equal(t, models.PlatformGithub, stored.Platform)
	require.Equal(t, "delivery-valid", stored.DeliveryID)
	require.Equal(t, "push", stored.Event)
	require.Equal(t, "opened", stored.Action)
	require.Equal(t, "goravel/framework", stored.Repository)
	require.Equal(t, "bowen", stored.Sender)
	require.Equal(t, body, stored.Payload)
}

func TestProcessWebhookUnknownEventTypeIsStored(t *testing.T) {
	env := newTestEnv(t)
	body := `{"action":"created","repository":{"full_name":"goravel/framework"},"sender":{"login":"bowen"}}`

	err := env.github.ProcessWebhook(signedRequest(env.secret, "custom_event_type", "delivery-unknown", body))

	require.NoError(t, err)
	require.Len(t, env.storedEvents, 1)
	require.Equal(t, "custom_event_type", env.storedEvents[0].Event)
	// No dispatch happened: no Discord call and no pull request lookup.
	require.Empty(t, env.discord.createThreadCalls)
	require.Empty(t, env.discord.sendMessageCalls)
	require.Empty(t, env.discord.deleteThreadCalls)
}

func TestProcessWebhookPullRequestOpened(t *testing.T) {
	env := newTestEnv(t)
	body := pullRequestBody(t, "opened")

	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-opened", body))

	require.NoError(t, err)

	// The event is stored first.
	require.Len(t, env.storedEvents, 1)
	require.Equal(t, "pull_request", env.storedEvents[0].Event)
	require.Equal(t, "opened", env.storedEvents[0].Action)

	// A thread is created in the configured channel using the payload title.
	require.Len(t, env.discord.createThreadCalls, 1)
	require.Equal(t, "pr-channel", env.discord.createThreadCalls[0].channelID)
	require.Equal(t, "Add tests", env.discord.createThreadCalls[0].thread.Title)

	// A pull request row is written with the created thread id.
	require.Len(t, env.pullRequestRows, 1)
	require.Equal(t, int64(123), env.pullRequestRows[0].GithubID)
	require.Equal(t, "thread-created", env.pullRequestRows[0].DiscordThreadID)
	require.Equal(t, "Add tests", env.pullRequestRows[0].Title)
}

func TestProcessWebhookPullRequestReadyForReview(t *testing.T) {
	env := newTestEnv(t)
	env.firstFn = populatePullRequestRow(t)

	body := pullRequestBody(t, "ready_for_review")
	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-ready", body))

	require.NoError(t, err)
	require.Empty(t, env.discord.createThreadCalls)
	require.Len(t, env.discord.sendMessageCalls, 1)
	require.Equal(t, "thread-existing", env.discord.sendMessageCalls[0].channelID)
	require.Contains(t, env.discord.sendMessageCalls[0].message, "core-role")
}

func TestProcessWebhookPullRequestClosed(t *testing.T) {
	env := newTestEnv(t)
	env.firstFn = populatePullRequestRow(t)

	body := pullRequestBody(t, "closed")
	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-closed", body))

	require.NoError(t, err)
	require.Equal(t, []string{"thread-existing"}, env.discord.deleteThreadCalls)
	require.Len(t, env.deleteCalls, 1)
	require.Len(t, env.deleteCalls[0], 1)
}

func TestProcessWebhookPullRequestUnknownActionWithoutRow(t *testing.T) {
	env := newTestEnv(t)
	body := pullRequestBody(t, "synchronize")

	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-sync", body))

	require.NoError(t, err)
	require.Empty(t, env.discord.createThreadCalls)
	require.Empty(t, env.discord.sendMessageCalls)
	require.Empty(t, env.discord.deleteThreadCalls)
	require.Empty(t, env.deleteCalls)
}

// A ready_for_review or closed delivery with no stored row must stop at the
// pullRequest.ID == 0 guard. Without it, either action would call Discord with
// an empty thread id, so removing the guard has to fail this test.
func TestProcessWebhookPullRequestWithoutRowDoesNotDispatch(t *testing.T) {
	for _, action := range []string{"ready_for_review", "closed"} {
		t.Run(action, func(t *testing.T) {
			env := newTestEnv(t)
			// The default fake First returns nil and leaves the row zero.
			body := pullRequestBody(t, action)

			err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-no-row", body))

			require.NoError(t, err)
			require.Empty(t, env.discord.createThreadCalls)
			require.Empty(t, env.discord.sendMessageCalls)
			require.Empty(t, env.discord.deleteThreadCalls)
			require.Empty(t, env.deleteCalls)
		})
	}
}

// A store failure on the known-event path (the storeEvent call after a
// successful parse) is logged and swallowed, and dispatch still runs.
func TestProcessWebhookKnownEventStoreFailureIsLogged(t *testing.T) {
	env := newTestEnv(t)
	storeErr := errors.New("db down")
	calls := 0
	env.firstOrCreateFn = func(dest any, conds ...any) error {
		calls++
		if calls == 1 {
			return storeErr
		}

		return nil
	}

	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-known-store-fail", pullRequestBody(t, "opened")))

	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Len(t, env.logs, 1)
	require.Contains(t, env.logs[0], "failed to store webhook event")
	require.Contains(t, env.logs[0], "db down")
	// Dispatch still runs even though the audit store failed.
	require.Len(t, env.discord.createThreadCalls, 1)
}

// A store failure on the unknown-event path (the storeEvent call inside the
// non-signature parse-error branch) is logged and swallowed, so the request is
// not failed just because the audit store failed.
func TestProcessWebhookUnknownEventStoreFailureIsLogged(t *testing.T) {
	env := newTestEnv(t)
	storeErr := errors.New("db down")
	env.firstOrCreateFn = func(dest any, conds ...any) error { return storeErr }
	body := `{"action":"created","repository":{"full_name":"goravel/discord"},"sender":{"login":"bowen"}}`

	err := env.github.ProcessWebhook(signedRequest(env.secret, "custom_event_type", "delivery-unknown-store-fail", body))

	require.NoError(t, err)
	require.Len(t, env.storedEvents, 1)
	require.Len(t, env.logs, 1)
	require.Contains(t, env.logs[0], "failed to store webhook event")
	require.Contains(t, env.logs[0], "db down")
}

// A failure writing the pull request row is returned to the caller. The event
// store runs first, so only the second FirstOrCreate (the row) may fail.
func TestProcessWebhookPullRequestRowFailureIsReturned(t *testing.T) {
	env := newTestEnv(t)
	rowErr := errors.New("db down")
	calls := 0
	env.firstOrCreateFn = func(dest any, conds ...any) error {
		calls++
		if calls == 2 {
			return rowErr
		}

		return nil
	}
	body := pullRequestBody(t, "opened")

	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-row-fail", body))

	require.ErrorIs(t, err, rowErr)
	require.Equal(t, 2, calls)
	// The thread is created before the failing row write.
	require.Len(t, env.discord.createThreadCalls, 1)
}

// Discord failures from the ready-for-review and closed handlers propagate out
// of ProcessWebhook; a failing thread delete must leave the row untouched so the
// thread-before-row ordering is pinned.
func TestProcessWebhookPullRequestDiscordErrors(t *testing.T) {
	t.Run("send message failure propagates", func(t *testing.T) {
		env := newTestEnv(t)
		env.firstFn = populatePullRequestRow(t)
		env.discord.sendMessageErr = errors.New("discord down")

		err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-send-fail", pullRequestBody(t, "ready_for_review")))

		require.ErrorIs(t, err, env.discord.sendMessageErr)
		require.Len(t, env.discord.sendMessageCalls, 1)
		require.Equal(t, "thread-existing", env.discord.sendMessageCalls[0].channelID)
	})

	t.Run("delete thread failure propagates and skips the row delete", func(t *testing.T) {
		env := newTestEnv(t)
		env.firstFn = populatePullRequestRow(t)
		env.discord.deleteThreadErr = errors.New("discord down")

		err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-delete-fail", pullRequestBody(t, "closed")))

		require.ErrorIs(t, err, env.discord.deleteThreadErr)
		require.Equal(t, []string{"thread-existing"}, env.discord.deleteThreadCalls)
		require.Empty(t, env.deleteCalls)
	})
}

// A failure deleting the row after the thread is removed is returned to the
// caller; the thread delete still happens first.
func TestProcessWebhookPullRequestRowDeleteFailureIsReturned(t *testing.T) {
	env := newTestEnv(t)
	env.firstFn = populatePullRequestRow(t)
	env.deleteErr = errors.New("db down")

	err := env.github.ProcessWebhook(signedRequest(env.secret, "pull_request", "pr-row-delete-fail", pullRequestBody(t, "closed")))

	require.ErrorIs(t, err, env.deleteErr)
	require.Equal(t, []string{"thread-existing"}, env.discord.deleteThreadCalls)
	require.Len(t, env.deleteCalls, 1)
}

func TestHandlePullRequestOpenedEventDraft(t *testing.T) {
	env := newTestEnv(t)
	payload := parsePullRequestPayload(t, pullRequestBody(t, "opened"))
	payload.PullRequest.Draft = true

	require.NoError(t, env.github.handlePullRequestOpenedEvent(payload))

	require.Len(t, env.discord.createThreadCalls, 1)
	require.Contains(t, env.discord.createThreadCalls[0].thread.Content, "Draft")
}

func TestHandlePullRequestOpenedEventCreateThreadError(t *testing.T) {
	env := newTestEnv(t)
	env.discord.createThreadErr = errors.New("discord down")

	err := env.github.handlePullRequestOpenedEvent(parsePullRequestPayload(t, pullRequestBody(t, "opened")))

	require.ErrorIs(t, err, env.discord.createThreadErr)
	require.Empty(t, env.pullRequestRows)
}

// TestHandlePullRequestLabeledEvent pins handlePullRequestLabeledEvent's
// behaviour. Its only production call site is the commented-out `case "labeled"`
// in ProcessWebhook, so no live webhook reaches it today; keeping the test means
// the handler is already covered if that case is re-enabled.
func TestHandlePullRequestLabeledEvent(t *testing.T) {
	tests := []struct {
		name        string
		labels      []string
		wantMessage bool
	}{
		{name: "review ready label sends message", labels: []string{"Review Ready"}, wantMessage: true},
		{name: "review ready substring sends message", labels: []string{"Needs Review Ready Soon"}, wantMessage: true},
		{name: "other label does not send message", labels: []string{"bug"}, wantMessage: false},
		{name: "no label does not send message", labels: nil, wantMessage: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t)
			pullRequest := &models.PullRequest{DiscordThreadID: "thread-labeled"}
			payload := parsePullRequestPayload(t, pullRequestBody(t, "labeled", test.labels...))

			require.NoError(t, env.github.handlePullRequestLabeledEvent(payload, pullRequest))

			if test.wantMessage {
				require.Len(t, env.discord.sendMessageCalls, 1)
				require.Equal(t, "thread-labeled", env.discord.sendMessageCalls[0].channelID)
			} else {
				require.Empty(t, env.discord.sendMessageCalls)
			}
		})
	}
}

func TestGetPullRequest(t *testing.T) {
	t.Run("returns a populated row when one exists", func(t *testing.T) {
		env := newTestEnv(t)
		env.firstFn = func(dest any) error {
			pullRequest, ok := dest.(*models.PullRequest)
			require.True(t, ok, "dest = %#v, want *models.PullRequest", dest)
			pullRequest.ID = 42
			pullRequest.GithubID = 123
			pullRequest.DiscordThreadID = "thread-42"
			pullRequest.Title = "Add tests"

			return nil
		}

		pullRequest, err := env.github.getPullRequest(123)

		require.NoError(t, err)
		require.Equal(t, uint(42), pullRequest.ID)
		require.Equal(t, int64(123), pullRequest.GithubID)
		require.Equal(t, "thread-42", pullRequest.DiscordThreadID)
		require.Equal(t, "Add tests", pullRequest.Title)

		require.Equal(t, []any{"github_id"}, env.whereQueries)
		require.Len(t, env.whereArgs, 1)
		require.Equal(t, []any{int64(123)}, env.whereArgs[0])
	})

	t.Run("returns a zero-value row when nothing is found", func(t *testing.T) {
		env := newTestEnv(t)

		pullRequest, err := env.github.getPullRequest(123)

		require.NoError(t, err)
		require.NotNil(t, pullRequest)
		require.Zero(t, pullRequest.ID)
	})

	t.Run("propagates a query error", func(t *testing.T) {
		env := newTestEnv(t)
		env.firstFn = func(dest any) error { return errors.New("db down") }

		pullRequest, err := env.github.getPullRequest(123)

		require.Error(t, err)
		require.Nil(t, pullRequest)
	})
}

func TestStoreEventUsesTrashedLookup(t *testing.T) {
	env := newTestEnv(t)
	body := `{"action":"opened"}`

	require.NoError(t, env.github.ProcessWebhook(signedRequest(env.secret, "push", "delivery-trashed", body)))

	require.Equal(t, 1, env.withTrashedCalls)

	require.Len(t, env.lookupEvents, 1)
	require.Equal(t, models.PlatformGithub, env.lookupEvents[0].Platform)
	require.Equal(t, "delivery-trashed", env.lookupEvents[0].DeliveryID)
}

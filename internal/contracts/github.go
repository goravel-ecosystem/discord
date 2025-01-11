package contracts

import "net/http"

type Github interface {
	ProcessWebhook(r *http.Request) error
}

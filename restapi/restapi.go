package restapi

import (
	"fmt"

	"github.com/unitychain/zkvote-node/restapi/controller"
	"github.com/unitychain/zkvote-node/zkvote"
)

type allOpts struct {
	webhookURLs  []string
	defaultLabel string
}

// Opt represents a REST Api option.
type Opt func(opts *allOpts)

// RESTAPI contains handlers for REST API
type RESTAPI struct {
	handlers []controller.Handler
}

// GetHandlers returns all controller REST API endpoints
func (c *RESTAPI) GetHandlers() []controller.Handler {
	return c.handlers
}

// NewRESTAPI returns new controller REST API instance.
func NewRESTAPI(node *zkvote.Node, opts ...Opt) (*RESTAPI, error) {
	restAPIOpts := &allOpts{}
	// Apply options
	for _, opt := range opts {
		opt(restAPIOpts)
	}

	var allHandlers []controller.Handler

	collectorController, err := controller.NewCollectorController(node)
	if err != nil {
		fmt.Print(err)
	}

	allHandlers = append(allHandlers, collectorController.GetRESTHandlers()...)

	return &RESTAPI{handlers: allHandlers}, nil
}

// // WithWebhookURLs is an option for setting up a webhook dispatcher which will notify clients of events
// func WithWebhookURLs(webhookURLs ...string) Opt {
// 	return func(opts *allOpts) {
// 		opts.webhookURLs = webhookURLs
// 	}
// }

// // WithDefaultLabel is an option allowing for the defaultLabel to be set.
// func WithDefaultLabel(defaultLabel string) Opt {
// 	return func(opts *allOpts) {
// 		opts.defaultLabel = defaultLabel
// 	}
// }
package scan

import "strings"

// Option defines optional configuration properties for a scan event dispatcher.
type Option[T Message] struct {
	handlers []Handler[T]
	services []string
	subName  string
}

// WithHandlers adds the list of optional handlers for a scan event dispatcher.
func WithHandlers[T Message](handlers ...Handler[T]) func(*Option[T]) {
	return func(o *Option[T]) {
		for _, h := range handlers {
			if h != nil {
				o.handlers = append(o.handlers, h)
			}
		}
	}
}

// WithEmulatedServices adds the list of services to use when generating emulated scan events.
//
// The default list of emulated services is defined by DefaultEmulatedServices.
func WithEmulatedServices[T Message](services ...string) func(*Option[T]) {
	return func(o *Option[T]) {
		for _, s := range services {
			if s = strings.TrimSpace(s); s != "" {
				o.services = append(o.services, strings.ToUpper(s))
			}
		}
	}
}

// WithSubscriptionName sets the subscription name to use when consuming pub/sub events.
//
// The default subscription name is DefaultSubscriptionName.
func WithSubscriptionName[T Message](name string) func(*Option[T]) {
	return func(o *Option[T]) {
		o.subName = strings.TrimSpace(name)
	}
}

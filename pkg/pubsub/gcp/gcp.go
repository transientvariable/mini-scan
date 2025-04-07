package gcp

import (
	gcpubsub "cloud.google.com/go/pubsub"
	gcdpubsub "gocloud.dev/pubsub"
)

// Message defines the constraints for message types.
type Message interface {
	*gcpubsub.Message | *gcdpubsub.Message | []byte | constr
}

// constr is used for compile-time assertion for types that handle the Message constraint.
type constr interface{}

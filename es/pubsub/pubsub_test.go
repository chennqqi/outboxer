package pubsub_test

import (
	"context"
	"testing"

	pubsubraw "cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/pubsub"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func TestPublishMessages(t *testing.T) {
	cases := []struct {
		name       string
		message    *outboxer.OutboxMessage
		shouldFail bool
	}{
		{
			name: "sending a message",
			message: &outboxer.OutboxMessage{
				Payload: []byte("hello world"),
				Options: map[string]interface{}{
					pubsub.TopicNameOption:   "test",
					pubsub.OrderingKeyOption: "",
				},
			},
			shouldFail: false,
		},
		{
			name: "sending a message to non existent topic",
			message: &outboxer.OutboxMessage{
				Payload: []byte("hello world"),
				Options: map[string]interface{}{
					pubsub.TopicNameOption:   "non-existent",
					pubsub.OrderingKeyOption: "",
				},
			},
			shouldFail: true,
		},
	}

	ctx := context.Background()

	srv := pstest.NewServer()
	defer srv.Close()

	conn, err := grpc.Dial(srv.Addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("an error was not expected: %s", err)
	}

	defer conn.Close()

	client, err := pubsubraw.NewClient(ctx, "project", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("an error was not expected: %s", err)
	}

	mustCreateTopic(ctx, t, client, "test")

	p := pubsub.New(client)

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			err = p.Send(ctx, c.message)
			if c.shouldFail {
				if err == nil {
					t.Fatalf("an error was expected: %s", err)
				}
			} else {
				if err != nil {
					t.Fatalf("an error was not expected: %s", err)
				}
			}
		})
	}
}

func mustCreateTopic(ctx context.Context, t *testing.T, pc *pubsubraw.Client, id string) *pubsubraw.Topic {
	top, err := pc.CreateTopic(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return top
}

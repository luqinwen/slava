package pubsub

import (
	"github.com/hdt3213/godis/datastruct/list"
	"reflect"
	"slava/internal/protocol"
	"slava/pkg/connection"
	"testing"
)

var (
	hub      = MakeHub()
	client   = connection.NewFakeConn()
	channels = [][]byte{[]byte("test1"), []byte("test2")}
)

func TestSubscribe(t *testing.T) {
	Subscribe(hub, client, channels)
	if !reflect.DeepEqual(client.GetChannels(), []string{"test1", "test2"}) {
		t.Error("Don't subscribe the correct channels")
	}

	raw, ok := hub.subs.Get("test1")
	if !ok {
		t.Error("subscriber list is not correct")
	} else {
		subscribers, _ := raw.(*list.LinkedList)
		if !subscribers.Contains(client) {
			t.Error("subscriber list is not correct")
		}
	}
}

func TestUnSubscribe(t *testing.T) {
	Subscribe(hub, client, channels)
	UnSubscribe(hub, client, [][]byte{[]byte("test1")})
	if !reflect.DeepEqual(client.GetChannels(), []string{"test2"}) {
		t.Error("Unsubscribe fail")
	}
}

func TestUnsubscribeAll(t *testing.T) {
	Subscribe(hub, client, channels)
	UnsubscribeAll(hub, client)
	if !reflect.DeepEqual(client.GetChannels(), []string{}) {
		t.Error("UnsubscribeAll fail")
	}
}

func TestPublish(t *testing.T) {
	Subscribe(hub, client, channels)
	arg1 := []byte("test1")
	arg2 := []byte("msg")
	if !reflect.DeepEqual(Publish(hub, [][]byte{arg1, arg2}), protocol.MakeIntReply(1)) {
		t.Error("Publish fail")
	}
}

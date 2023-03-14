package pubsub

import (
	"github.com/hdt3213/godis/datastruct/list"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/internal/utils"
	"strconv"
)

var (
	_subscribe         = "subscribe"
	_unsubscribe       = "unsubscribe"
	messageBytes       = []byte("message")
	unSubscribeNothing = []byte("*3\r\n$11\r\nunsubscribe\r\n$-1\n:0\r\n")
)

func makeMsg(t string, channel string, code int64) []byte {
	return []byte("*3\r\n$" + strconv.FormatInt(int64(len(t)), 10) + protocol.CRLF + t + protocol.CRLF +
		"$" + strconv.FormatInt(int64(len(channel)), 10) + protocol.CRLF + channel + protocol.CRLF +
		":" + strconv.FormatInt(code, 10) + protocol.CRLF)
}

/*
 * invoker should lock channel
 * return: is new subscribed
 */
func subscribe0(hub *Hub, channel string, client slava.Connection) bool {
	client.Subscribe(channel)

	// add into hub.subs
	raw, ok := hub.subs.Get(channel)
	// If the channel is subscribed by other clients, get the list of its subscribers
	// Otherwise create a list to store the subscribers of this channel, then add them to hub.subs
	var subscribers *list.LinkedList
	if ok {
		subscribers, _ = raw.(*list.LinkedList)
	} else {
		subscribers = list.Make()
		hub.subs.Put(channel, subscribers)
	}
	// If the client has subscribed this channel before, return false
	if subscribers.Contains(func(a interface{}) bool {
		return a == client
	}) {
		return false
	}
	// If the client has not subscribed this channel before, add it to the list
	subscribers.Add(client)
	return true
}

/*
 * invoker should lock channel
 * return: is actually un-subscribe
 */
func unsubscribe0(hub *Hub, channel string, client slava.Connection) bool {
	client.UnSubscribe(channel)

	// remove from hub.subs
	raw, ok := hub.subs.Get(channel)
	// Remove the client from the subscriber list of this channel
	if ok {
		subscribers, _ := raw.(*list.LinkedList)
		subscribers.RemoveAllByVal(func(a interface{}) bool {
			return utils.Equals(a, client)
		})

		if subscribers.Len() == 0 {
			// clean
			hub.subs.Remove(channel)
		}
		return true
	}
	// Fail to unsubscribe
	return false
}

// Subscribe puts the given connection into the given channel
func Subscribe(hub *Hub, c slava.Connection, args [][]byte) slava.Reply {
	channels := make([]string, len(args))
	for i, b := range args {
		channels[i] = string(b)
	}

	hub.subsLocker.Locks(channels...)
	defer hub.subsLocker.UnLocks(channels...)
	// If the channel is new subscribed, print the related information
	for _, channel := range channels {
		if subscribe0(hub, channel, c) {
			_, _ = c.Write(makeMsg(_subscribe, channel, int64(c.SubsCount())))
		}
	}
	return &protocol.NoReply{}
}

// UnsubscribeAll removes the given connection from all subscribing channel
func UnsubscribeAll(hub *Hub, c slava.Connection) {
	channels := c.GetChannels()

	hub.subsLocker.Locks(channels...)
	defer hub.subsLocker.UnLocks(channels...)
	// remove the connection from all the channel subscribers list
	for _, channel := range channels {
		unsubscribe0(hub, channel, c)
	}

}

// UnSubscribe removes the given connection from the given channel
func UnSubscribe(db *Hub, c slava.Connection, args [][]byte) slava.Reply {
	var channels []string
	// The only difference for UnsubscribeAll is self choosing the unsubscribed channels
	if len(args) > 0 {
		channels = make([]string, len(args))
		for i, b := range args {
			channels[i] = string(b)
		}
		// No channels found in the input, same to UnsubscribeAll
	} else {
		channels = c.GetChannels()
	}

	db.subsLocker.Locks(channels...)
	defer db.subsLocker.UnLocks(channels...)

	if len(channels) == 0 {
		_, _ = c.Write(unSubscribeNothing)
		return &protocol.NoReply{}
	}
	// If the channel is unsubscribed successfully, print the related information
	for _, channel := range channels {
		if unsubscribe0(db, channel, c) {
			_, _ = c.Write(makeMsg(_unsubscribe, channel, int64(c.SubsCount())))
		}
	}
	return &protocol.NoReply{}
}

// Publish send msg to all subscribing client
func Publish(hub *Hub, args [][]byte) slava.Reply {
	if len(args) != 2 {
		return &protocol.ArgNumErrReply{Cmd: "publish"}
	}
	channel := string(args[0])
	message := args[1]

	hub.subsLocker.Lock(channel)
	defer hub.subsLocker.UnLock(channel)

	raw, ok := hub.subs.Get(channel)
	if !ok {
		return protocol.MakeIntReply(0)
	}
	// Find the subscribers of the channel in the sub
	subscribers, _ := raw.(*list.LinkedList)
	// The client sends the corresponding message to every subscriber
	subscribers.ForEach(func(i int, c interface{}) bool {
		client, _ := c.(slava.Connection)
		replyArgs := make([][]byte, 3)
		replyArgs[0] = messageBytes
		replyArgs[1] = []byte(channel)
		replyArgs[2] = message
		_, _ = client.Write(protocol.MakeMultiBulkReply(replyArgs).ToBytes())
		return true
	})
	return protocol.MakeIntReply(int64(subscribers.Len()))
}

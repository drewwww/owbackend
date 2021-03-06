// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package rpc

import (
	"flag"
	"net/rpc"

	"github.com/Sirupsen/logrus"
	"github.com/TF2Stadium/Helen/config"
	"github.com/streadway/amqp"
	"github.com/vibhavp/amqp-rpc"
)

var (
	pauling   *rpc.Client
	fumble    *rpc.Client
	twitchbot *rpc.Client

	paulingDisabled   = flag.Bool("disable_pauling", true, "disable pauling")
	fumbleDisabled    = flag.Bool("disable_fumble", true, "disable fumble")
	twitchbotDisabled = flag.Bool("disable_twitchbot", true, "disable twitch bot")
)

func ConnectRPC(amqpConn *amqp.Connection) {
	if !*paulingDisabled {
		codec, err := amqprpc.NewClientCodec(amqpConn, config.Constants.PaulingQueue, amqprpc.JSONCodec{})
		if err != nil {
			logrus.Fatal(err)
		}

		pauling = rpc.NewClientWithCodec(codec)

	}
	if !*fumbleDisabled {
		codec, err := amqprpc.NewClientCodec(amqpConn, config.Constants.FumbleQueue, amqprpc.JSONCodec{})
		if err != nil {
			logrus.Fatal(err)
		}

		fumble = rpc.NewClientWithCodec(codec)
	}
	if !*twitchbotDisabled {
		codec, err := amqprpc.NewClientCodec(amqpConn, config.Constants.TwitchBotQueue, amqprpc.JSONCodec{})
		if err != nil {
			logrus.Fatal(err)
		}
		twitchbot = rpc.NewClientWithCodec(codec)
	}
}

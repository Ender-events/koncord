package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Variables used for command line parameters
type Bot struct {
	kube    *kubernetes.Clientset
	channel map[string]struct{}
}

var (
	token      string
	namespace  string
	deployment string
	bot        *Bot
)

func init() {
	token = os.Getenv("TOKEN")
	if token == "" {
		panic("Missing TOKEN env var")
	}
	namespace = os.Getenv("NAMESPACE")
	if token == "" {
		panic("Missing NAMESPACE env var")
	}
	deployment = os.Getenv("DEPLOYMENT")
	if token == "" {
		panic("Missing DEPLOYMENT env var")
	}
}

func NewBot() *Bot {
	var bot Bot
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	bot.channel = map[string]struct{}{}
	bot.kube, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return &bot
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	bot = NewBot()

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

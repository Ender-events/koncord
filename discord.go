package main

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (bot *Bot) addChannel(channelID string) {
	bot.channel[channelID] = struct{}{}
	fmt.Println(channelID)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	bot.addChannel(m.ChannelID)
	fmt.Printf("%s: %s\n", m.ChannelID, m.ContentWithMentionsReplaced())
	if m.ContentWithMentionsReplaced() == "@gameserver help" {
		s.ChannelMessageSend(m.ChannelID, "@gameserver help: this help command\n"+
			"@gameserver get pods: list all pods status\n"+
			"@gameserver restart: delete the pods to restart the server")
	}
	if m.ContentWithMentionsReplaced() == "@gameserver get pods" {
		pods := bot.getPodsFromDeployement(context.TODO(), namespace, deployment)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("nb pods: %d", len(pods)))
		for _, pod := range pods {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%v", pod.Status))
		}
	}

	if m.ContentWithMentionsReplaced() == "@gameserver restart" {
		pods := bot.getPodsFromDeployement(context.TODO(), namespace, deployment)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Restarting %d pods", len(pods)))
		for _, pod := range pods {
			err := bot.kube.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %s", err.Error()))
			}
		}
		s.ChannelMessageSend(m.ChannelID, "Done")
	}
}

package main

import (
	"strings"

	ai "github.com/sashabaranov/go-openai"
	vip "github.com/spf13/viper"
)

func splitResponse(response string, maxLineLength int) []string {
	words := strings.Fields(response)
	messages := []string{}
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 > maxLineLength {
			messages = append(messages, currentLine)
			currentLine = ""
		}
		if len(currentLine) > 0 {
			currentLine += " "
		}
		currentLine += word
	}

	if currentLine != "" {
		messages = append(messages, currentLine)
	}

	return messages
}

// util
func isAdmin(nick string) bool {
	admins := vip.GetStringSlice("admins")
	if len(admins) == 0 {
		return true
	}
	for _, user := range admins {
		if user == nick {
			return true
		}
	}
	return false
}

func sumMessageLengths(messages []ai.ChatCompletionMessage) int {
	sum := 0
	for _, m := range messages {
		sum += len(m.Content)
	}
	return sum
}

func keysAsString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

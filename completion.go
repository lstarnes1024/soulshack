package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"time"

	ai "github.com/sashabaranov/go-openai"
)

func ChatCompletionTask(ctx *ChatContext) <-chan *string {
	ch := make(chan *string)
	go chatCompletionStream(ctx, ch)
	return ch
}

func chatCompletionStream(cc *ChatContext, channel chan<- *string) {

	defer close(channel)
	cc.Stats()

	ctx, cancel := context.WithTimeout(cc, cc.Session.Config.ClientTimeout)
	defer cancel()

	stream, err := cc.AI.CreateChatCompletionStream(ctx, ai.ChatCompletionRequest{
		MaxTokens: cc.Session.Config.MaxTokens,
		Model:     cc.Personality.Model,
		Messages:  cc.Session.GetHistory(),
		Stream:    true,
	})

	if err != nil {
		senderror(err, channel)
		return
	}

	defer stream.Close()
	chunker := &Chunker{
		Size:     cc.Session.Config.Chunkmax,
		Last:     time.Now(),
		Boundary: boundary,
		Timeout:  cc.Session.Config.Chunkdelay,
	}

	for {
		response, err := stream.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				senderror(err, channel)
			}
			send(chunker.Buffer.String(), channel)
			return
		}
		if len(response.Choices) != 0 {
			chunker.Buffer.WriteString(response.Choices[0].Delta.Content)
		}
		for {
			if ready, chunk := chunker.Chunk(); ready {
				send(chunk, channel)
			} else {
				break
			}
		}
	}
}

func senderror(err error, channel chan<- *string) {
	e := err.Error()
	channel <- &e
}

func send(chunk string, channel chan<- *string) {
	channel <- &chunk
}

type Chunker struct {
	Size     int
	Last     time.Time
	Buffer   bytes.Buffer
	Boundary regexp.Regexp
	Timeout  time.Duration
}

var boundary = *regexp.MustCompile(`(?m)[.:!?]\s`)

func (c *Chunker) Chunk() (bool, string) {

	// chunk if n seconds have passed since the last chunk
	if time.Since(c.Last) >= c.Timeout {
		content := c.Buffer.String()
		indices := c.Boundary.FindAllStringIndex(content, -1)
		if len(indices) > 0 {
			last := indices[len(indices)-1]
			chunk := c.Buffer.Next(last[1])
			c.Last = time.Now()
			return true, string(chunk)
		}
	}

	// always chunk on a newline in the buffer
	index := bytes.IndexByte(c.Buffer.Bytes(), '\n')
	if index != -1 && index < c.Size {
		chunk := c.Buffer.Next(index + 1)
		c.Last = time.Now()
		return true, string(chunk)
	}

	// chunk if full buffer satisfies chunk size
	if c.Buffer.Len() >= c.Size {
		chunk := c.Buffer.Next(c.Size)
		c.Last = time.Now()
		return true, string(chunk)
	}

	return false, ""

}

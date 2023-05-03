package telegramist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"i3_stat/internal/entities"
	"i3_stat/internal/utils"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Sender struct {
	infMsgList   map[uint32]entities.Message
	infMsgListMU sync.Mutex
	chat         string
	token        string
	apiURL       string
}

func NewSender(token, chat string) *Sender {
	return &Sender{
		chat:       chat,
		token:      token,
		apiURL:     "https://api.telegram.org/bot" + token + "/sendMessage",
		infMsgList: make(map[uint32]entities.Message, 100),
	}
}

func (s *Sender) InstantMessage(message string) {
	s.sendTelegram(s.chat, message)
}

func (s *Sender) HandleMessage(message entities.Message) {
	message.Time = time.Now()
	hash := utils.HashAny(message.Messages)

	s.infMsgListMU.Lock()
	if tmp, ok := s.infMsgList[hash]; ok {
		tmp.Total++
		s.infMsgList[hash] = tmp
	} else {
		message.Total = 1
		s.infMsgList[hash] = message
	}
	s.infMsgListMU.Unlock()
}

func (s *Sender) Start() {
	go func() {
		for range time.NewTicker(30 * time.Second).C {
			go s.processInfo()
		}
	}()
}

func (s *Sender) processInfo() {
	s.infMsgListMU.Lock()
	defer s.infMsgListMU.Unlock()
	if len(s.infMsgList) == 0 {
		return
	}
	msgList := s.infMsgList
	s.infMsgList = make(map[uint32]entities.Message, 100)
	go s.send(s.chat, msgList)
}

func (s *Sender) send(chat string, messages map[uint32]entities.Message) {
	for _, message := range s.prepareMessages(messages) {
		s.sendTelegram(chat, message)
	}
}

func (s *Sender) sendTelegram(chat, message string) {
	requestBody, _ := json.Marshal(map[string]string{
		"chat_id": chat,
		"text":    "```" + message + "```",
	})
	body, err := http.Post(s.apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Error while sending message to telegram", err)
		return
	}
	if err = body.Body.Close(); err != nil {
		log.Println("Error while closing response body", err)
	}
}

func (s *Sender) prepareMessages(messages map[uint32]entities.Message) []string {
	var result [][]string
	totalCount := 0
	batchMessages := make([]string, 0, len(messages))
	for _, message := range messages {
		msg := fmt.Sprintf("%s \n%s", message.Time.Format("2006-01-02 15:04:05"), strings.Join(message.Messages, "\n"))
		if message.Total > 1 {
			msg = fmt.Sprintf("%s (%d) \n%s", message.Time.Format("2006-01-02 15:04:05"), message.Total, strings.Join(message.Messages, "\n"))
		}
		message.Total = 0
		totalCount += len(msg)
		if totalCount > 4096 {
			if len(batchMessages) > 0 {
				result = append(result, batchMessages)
				batchMessages = make([]string, 0, len(messages))
			}
			totalCount = len(msg)
		}
		batchMessages = append(batchMessages, msg)
	}
	if len(batchMessages) > 0 {
		result = append(result, batchMessages)
	}
	if len(result) == 0 {
		return nil
	}
	var finalResult []string
	for _, batch := range result {
		finalResult = append(finalResult, strings.Join(batch, "\n"))
	}
	return finalResult
}

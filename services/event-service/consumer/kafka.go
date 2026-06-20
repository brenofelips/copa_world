package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"event-service/db"
	"event-service/models"
)

type Consumer struct {
	reader *kafka.Reader
	store  *db.Store
}

func New(brokers, groupID string, store *db.Store) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(brokers, ","),
		Topic:          "match-events",
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &Consumer{reader: r, store: store}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	log.Println("Event consumer running...")

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("event consumer: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var event models.NormalizedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("event consumer: unmarshal error: %v", err)
			continue
		}

		if err := c.store.SaveEvent(ctx, &event); err != nil {
			log.Printf("event consumer: save event %s error: %v", event.EventID, err)
		}
	}
}

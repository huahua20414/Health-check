package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type RawMessage struct {
	Topic      string
	Payload    []byte
	ReceivedAt time.Time
	Duplicate  bool
}

type DeviceEvent struct {
	Code       string                 `json:"code"`
	Timestamp  int64                  `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
	MessageKey string                 `json:"-"`
}

type Deduper struct {
	mu   sync.Mutex
	data map[string]time.Time
	ttl  time.Duration
}

func NewDeduper(ttl time.Duration) *Deduper {
	return &Deduper{
		data: make(map[string]time.Time),
		ttl:  ttl,
	}
}

func (d *Deduper) Seen(key string) bool {
	now := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	for k, v := range d.data {
		if now.Sub(v) > d.ttl {
			delete(d.data, k)
		}
	}

	if _, ok := d.data[key]; ok {
		return true
	}

	d.data[key] = now
	return false
}

type Pipeline struct {
	rawChan       chan RawMessage
	parsedChan    chan DeviceEvent
	retryChan     chan RawMessage
	deduper       *Deduper
	parseWorkers  int
	storeWorkers  int
	maxRetry      int
	retryInterval time.Duration
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		rawChan:       make(chan RawMessage, 2000),
		parsedChan:    make(chan DeviceEvent, 2000),
		retryChan:     make(chan RawMessage, 1000),
		deduper:       NewDeduper(10 * time.Minute),
		parseWorkers:  8,
		storeWorkers:  16,
		maxRetry:      3,
		retryInterval: 2 * time.Second,
	}
}

func (p *Pipeline) Start(ctx context.Context) {
	for i := 0; i < p.parseWorkers; i++ {
		go p.parseWorker(ctx)
	}
	for i := 0; i < p.storeWorkers; i++ {
		go p.storeWorker(ctx)
	}
	go p.retryWorker(ctx)
}

func (p *Pipeline) Ingest(msg RawMessage) error {
	select {
	case p.rawChan <- msg:
		return nil
	default:
		return fmt.Errorf("raw channel full")
	}
}

func (p *Pipeline) parseWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-p.rawChan:
			ev, err := p.parse(msg)
			if err != nil {
				log.Printf("parse failed topic=%s err=%v", msg.Topic, err)
				p.enqueueRetry(msg)
				continue
			}
			select {
			case p.parsedChan <- ev:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (p *Pipeline) storeWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-p.parsedChan:
			if p.deduper.Seen(ev.MessageKey) {
				log.Printf("duplicate dropped key=%s", ev.MessageKey)
				continue
			}
			if err := p.store(ev); err != nil {
				log.Printf("store failed key=%s err=%v", ev.MessageKey, err)
			}
		}
	}
}

func (p *Pipeline) retryWorker(ctx context.Context) {
	type retryItem struct {
		msg   RawMessage
		count int
	}
	queue := make(chan retryItem, 1000)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-p.retryChan:
				queue <- retryItem{msg: msg, count: 1}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case item := <-queue:
			time.Sleep(p.retryInterval)
			ev, err := p.parse(item.msg)
			if err != nil {
				if item.count < p.maxRetry {
					queue <- retryItem{msg: item.msg, count: item.count + 1}
				} else {
					log.Printf("retry exhausted topic=%s", item.msg.Topic)
				}
				continue
			}
			select {
			case p.parsedChan <- ev:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (p *Pipeline) enqueueRetry(msg RawMessage) {
	select {
	case p.retryChan <- msg:
	default:
		log.Printf("retry channel full topic=%s", msg.Topic)
	}
}

func (p *Pipeline) parse(msg RawMessage) (DeviceEvent, error) {
	var ev DeviceEvent
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return DeviceEvent{}, err
	}
	ev.MessageKey = buildMessageKey(msg.Topic, msg.Payload)
	return ev, nil
}

func (p *Pipeline) store(ev DeviceEvent) error {
	time.Sleep(10 * time.Millisecond)
	log.Printf("stored code=%s ts=%d key=%s", ev.Code, ev.Timestamp, ev.MessageKey)
	return nil
}

func buildMessageKey(topic string, payload []byte) string {
	h := sha1.New()
	h.Write([]byte(topic))
	h.Write([]byte(":"))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pipeline := NewPipeline()
	pipeline.Start(ctx)

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://127.0.0.1:1883")
	opts.SetClientID("consumer-qos1-pipeline")
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetOrderMatters(false)

	opts.OnConnect = func(c mqtt.Client) {
		token := c.Subscribe("devices/+/data", 1, func(client mqtt.Client, msg mqtt.Message) {
			payloadCopy := append([]byte(nil), msg.Payload()...)
			raw := RawMessage{
				Topic:      msg.Topic(),
				Payload:    payloadCopy,
				ReceivedAt: time.Now(),
				Duplicate:  msg.Duplicate(),
			}

			if err := pipeline.Ingest(raw); err != nil {
				log.Printf("ingest failed topic=%s err=%v", msg.Topic(), err)
			}
		})
		if token.Wait() && token.Error() != nil {
			log.Printf("subscribe failed: %v", token.Error())
		}
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	select {}
}

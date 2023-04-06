package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ReconnectTry = 3
)
const (
	ExKindFanout = "fanout"
	ExKindTopic  = "topic"
	ExKindDirect = "direct"
)

type DataReporter interface {
}
type ConsumeCallBackFunc func(data []byte) error

func SimpleCallback(data []byte) error {
	fmt.Println(string(data))
	time.Sleep(1 * time.Second)
	return nil
}

func FailOnError(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s:%s", msg, err))
	}
}

type PubMessage struct {
	Pub      *amqp.Publishing
	Exchange string
	Key      string
}

type RabbitMQ struct {
	conn     *amqp.Connection
	defChan  *amqp.Channel
	cxt      context.Context
	cancel   context.CancelFunc
	uri      string
	prefetch int
}

func NewRabbitMQ(cnf *RabbitMQConfig) *RabbitMQ {
	uri := cnf.GetUri()
	conn, err := amqp.Dial(uri)
	if err != nil {
		panic("连接rabbitmq失败,err:" + err.Error())
	}
	ch, err := conn.Channel()
	if err != nil {
		panic("获取channel失败,err:" + err.Error())
	}
	err = ch.Qos(cnf.PrefetchCount, 0, false)
	if err != nil {
		panic("设置channel的Qos失败,err:" + err.Error())
	}
	cxt, cancel := context.WithCancel(context.Background())
	mq := RabbitMQ{
		conn:     conn,
		defChan:  ch,
		cxt:      cxt,
		cancel:   cancel,
		uri:      uri,
		prefetch: cnf.PrefetchCount,
	}

	return &mq
}

// Publish 发布消息
func (r *RabbitMQ) Publish(exchange, key string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("rabbitmq publish json marshal error:", err)
		return err
	}
	pub := amqp.Publishing{
		ContentType:  "text/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	}
	return r.defChan.PublishWithContext(context.TODO(), exchange, key, false, false, pub)
}

type CallbackPubMsg struct {
	BackExchange string `json:"back_exchange"`
	BackQueue    string `json:"back_queue"`
	Content      string `json:"content"`
}

// SyncCallBackMsg 提供需要信息接收放同步返回信息的功能
// toExchange 目标邮局  toKey目标路由key  msg是消息内容
// useDefExchange 是否使用默认邮局""  timeout是同步等待最长时间
func (r *RabbitMQ) SyncCallBackMsg(toExchange, toKey, msg string, useDefExchange bool, timeout int) ([]byte, error) {
	tmpExchange := ""
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(1000)
	if !useDefExchange {
		tmpExchange = fmt.Sprintf("prootocol.tmp.ex_%d_%d", time.Now().Unix(), num)
	}
	tmpQueue := fmt.Sprintf("prootocol.tmp.qu_%d_%d", time.Now().Unix(), num)
	pubMsg := &CallbackPubMsg{
		BackExchange: tmpExchange,
		BackQueue:    tmpQueue,
		Content:      msg,
	}
	err := r.DeclareTmpQueue(pubMsg.BackExchange, pubMsg.BackQueue)
	if err != nil {
		fmt.Println("生成临时队列失败")
		return nil, err
	}
	data, err := json.Marshal(pubMsg)
	if err != nil {
		return nil, err
	}
	pub := amqp.Publishing{
		ContentType: "text/json", //"text/plain"
		Body:        data,
	}
	time.Sleep(10 * time.Second)
	err = r.defChan.PublishWithContext(
		context.TODO(),
		toExchange,
		toKey,
		false,
		false,
		pub,
	)
	if err != nil {
		return nil, err
	}
	msgChan, err := r.defChan.Consume(pubMsg.BackQueue, "sync_callback", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	c := time.After(time.Duration(timeout) * time.Second)
	select {
	case msg := <-msgChan:
		_ = msg.Ack(false)
		return msg.Body, nil
	case <-c:
		return nil, errors.New("mq callback timeout")
	}
}

// GetMsg 获取单条信息
func (r *RabbitMQ) GetMsg(queue string) ([]byte, error) {
	msg, ok, err := r.defChan.Get(queue, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("ch get error")
	}
	defer func() {
		_ = r.defChan.Ack(msg.DeliveryTag, false)
	}()
	return msg.Body, nil
}

// Consume 消费信息
func (r *RabbitMQ) Consume(queue string, fn ConsumeCallBackFunc) error {
	msgChan, err := r.defChan.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		fmt.Println("channel consume failed: ", err)
		return err
	}
	for {
		select {
		case msg, ok := <-msgChan:
			if ok {
				//log.Printf("[x] %s, %s \n", d.RoutingKey, d.Body)
				err = fn(msg.Body)
				if err == nil {
					_ = msg.Ack(false)
				} else {
					fmt.Println("callback function run error:", err)
				}

			} else {
				fmt.Println("mq channel is closed")
				return amqp.ErrClosed
			}

		case <-r.cxt.Done():
			fmt.Println("service is closing")
			return nil

		}
	}

}

func (r *RabbitMQ) DeclareDirectEx(exchange string) error {
	return r.declareExchange(exchange, ExKindDirect)
}

func (r *RabbitMQ) DeclareFanoutEx(exchange string) error {
	return r.declareExchange(exchange, ExKindFanout)
}

func (r *RabbitMQ) DeclareTopicEx(exchange string) error {
	return r.declareExchange(exchange, ExKindTopic)
}

func (r *RabbitMQ) declareExchange(exchange, kind string) error {

	return r.defChan.ExchangeDeclare(
		exchange,
		kind,
		true,
		false,
		false,
		false,
		nil,
	)
}

// DeclareBindQueue 声明和绑定队列
func (r *RabbitMQ) DeclareBindQueue(queue, exchange, key string) error {
	if queue == "" {
		return errors.New("queue or ch or exchange is null")
	}

	_, err := r.defChan.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}
	if exchange != "" {
		return r.defChan.QueueBind(queue, key, exchange, false, nil)
	}
	return nil
}

// DeclareTmpQueue 创建临时直联队列和邮局
func (r *RabbitMQ) DeclareTmpQueue(tmpExchange, tmpQueue string) error {
	if tmpQueue == "" {
		return errors.New("queue is null")
	}
	_, err := r.defChan.QueueDeclare(
		tmpQueue,
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		fmt.Println("queue declare error")
		return err
	}

	if tmpExchange != "" {
		err := r.defChan.ExchangeDeclare(
			tmpExchange,
			ExKindDirect,
			false,
			true,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println("exchange declare error")
			return err
		}
	}
	if tmpExchange != "" {
		err := r.defChan.QueueBind(tmpQueue, tmpQueue, tmpExchange, false, nil)
		if err != nil {
			fmt.Println("queue bind exchange error")
			return err
		}
	}
	return nil
}

// GetChannel 获取新的通道
func (r *RabbitMQ) GetChannel() (*amqp.Channel, error) {
	if r.conn.IsClosed() {
		if !r.ReConnect() {
			fmt.Println("rabbit mq is not connected")
			return nil, amqp.ErrClosed
		}
	}
	ch, err := r.conn.Channel()
	if err != nil {
		fmt.Println("获取channel失败")
		return nil, err
	}
	err = ch.Qos(r.prefetch, 0, false)
	if err != nil {
		fmt.Println("设置channel的Qos失败")
		return nil, err
	}
	return ch, nil
}

// ReConnect 重连
func (r *RabbitMQ) ReConnect() bool {
	i := 0
	for {
		i++
		if i >= ReconnectTry {
			return false
		}
		conn, err := amqp.Dial(r.uri)
		if err == nil {
			ch, err := r.conn.Channel()
			if err != nil {
				fmt.Println("获取channel失败")
				time.Sleep(1 * time.Second)
				continue
			}
			err = ch.Qos(r.prefetch, 0, false)
			if err != nil {
				fmt.Println("设置channel的Qos失败")
				time.Sleep(1 * time.Second)
				continue
			}
			r.conn = conn
			r.defChan = ch
			break
		} else {
			fmt.Println("连接rabbitmq失败")
			time.Sleep(1 * time.Second)
			continue
		}
	}

	return true
}

// IsConnect 连接是否正常
func (r *RabbitMQ) IsConnect() bool {
	return !r.conn.IsClosed()
}

func (r *RabbitMQ) Close() {
	r.cancel()
	_ = r.defChan.Close()
	_ = r.conn.Close()
}

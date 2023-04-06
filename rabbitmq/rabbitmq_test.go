package rabbitmq

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	eReportData = "protocol.v1.report"
	qDeviceData = "protocol.v1.report.data"
)

func getMQ() *RabbitMQ {
	cnf := &RabbitMQConfig{
		Host:          "127.0.0.1",
		Port:          "5672",
		Username:      "develop",
		Password:      "develop",
		VirtualHost:   "dev",
		PrefetchCount: 5,
	}

	return NewRabbitMQ(cnf)
}

func TestRabbitMQ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	rbmq := getMQ()

	err := rbmq.DeclareFanoutEx(eReportData)
	if err != nil {
		t.Error(err)
		return
	}
	err = rbmq.DeclareBindQueue(qDeviceData, eReportData, "")

	if err != nil {
		t.Error(err)
		return
	}
	err = rbmq.Publish(eReportData, "", "test,ddd")
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(1 * time.Second)
	ch, err := rbmq.GetChannel()
	if err != nil {
		t.Error("channel should be nil", err)
		return
	}
	defer ch.Close()
	q, _ := ch.QueueInspect(qDeviceData)
	t.Log(q.Name, q.Messages, q.Consumers)
	go func() {
		time.Sleep(2 * time.Second)
		rbmq.Close()
	}()
	//ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err = rbmq.Consume("test.logs", SimpleCallback)
	if err != nil && err != amqp.ErrClosed {
		t.Error(err)
		return
	}
	t.Log("ok")

}

func TestBindKey(t *testing.T) {
	//if testing.Short() {
	//	t.Skip("skipping test in short mode")
	//}
	rbmq := getMQ()

	err := rbmq.DeclareTopicEx(eReportData)
	if err != nil {
		t.Error(err)
		return
	}

	err = rbmq.DeclareBindQueue("protocol.v1.report.device.data", eReportData, "protocol.v1.report.device.data.#")
	if err != nil {
		t.Error(err)
		return
	}

	//err = rbmq.DeclareBindQueue("protocol.v1.report.device.alarm", eReportData, "protocol.v1.report.device.alarm")
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	err = rbmq.DeclareBindQueue("protocol.v1.report.device", eReportData, "protocol.v1.report.device.#")
	if err != nil {
		t.Error(err)
		return
	}
	err = rbmq.Publish(eReportData, "protocol.v1.report.device.alarm", fmt.Sprintf("test alarm message %v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
		return
	}
	//err = rbmq.Publish(eReportData, "protocol.v1.report.device.data.env", fmt.Sprintf("test env data message %v", 3889))
	//if err != nil {
	//	t.Error(err)
	//	return
	//}

}

func TestRabbitMQ_Get(t *testing.T) {
	mq := getMQ()
	defer mq.Close()
	//ch, err := mq.GetChannel()
	//if err != nil {
	//	t.Error("channel should be nil", err)
	//	return
	//}
	//defer ch.Close()
	tests := []struct {
		name    string
		queue   string
		wantErr bool
	}{
		{"队列有消息", "test.logs", false},
		{"队列无消息", "test.direct", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err1 := mq.GetMsg(tt.queue)

			if tt.wantErr != (err1 != nil) {
				t.Error("get msg err:", err1)
				return
			}
			if err1 == nil {
				t.Log("received:", string(msg))
			}

		})

	}

}

func TestRabbitMQ_DirectExchange(t *testing.T) {
	mq := getMQ()
	defer mq.Close()
	//ch, err := mq.GetChannel()
	//if err != nil {
	//	t.Error("channel should be nil", err)
	//	return
	//}
	//defer ch.Close()
	exName := "protocol.direct"
	err := mq.DeclareDirectEx(exName)
	if err != nil {
		t.Error("declare direct exchange err:", err)
		return
	}
	err = mq.DeclareBindQueue("test.direct", exName, "test.direct")
	if err != nil {
		t.Error("declare queue and bind queue err:", err)
		return
	}

	_ = mq.Publish(exName, "test.direct", "direct01")
	queueName := "protocol.direct.dispatch.task"
	err = mq.DeclareBindQueue(queueName, exName, queueName)
	if err != nil {
		t.Error("declare queue and bind queue err:", err)
		return
	}
	_ = mq.Publish(exName, queueName, "direct02")

}

func TestRabbitMQ_Consume(t *testing.T) {
	t.Parallel()
	mq := getMQ()
	//defer mq.Close()
	//ch, err := mq.GetChannel()
	//if err != nil {
	//	t.Error("channel should be nil", err)
	//	return
	//}
	//defer ch.Close()
	//exName := "protocol.direct"
	//queueName := "protocol.direct.dispatch.task"
	callback := func(msg []byte) error {
		fmt.Println("received:", string(msg))
		var cpMsg CallbackPubMsg
		err := json.Unmarshal(msg, &cpMsg)
		if err != nil {
			fmt.Println("decode callback pub msg:", err)
			return err
		}
		//time.Sleep(5 * time.Second)
		err = mq.Publish(cpMsg.BackExchange, cpMsg.BackQueue, "i receive, send back ")
		if err != nil {
			fmt.Println("publish callback msg err:", err)
			return err
		}
		//fmt.Println("-----waiting----")
		return nil
	}
	go func() {
		time.Sleep(15 * time.Second)
		mq.Close()
	}()

	err := mq.Consume("test.pub", callback)
	if err != nil {
		t.Error("consume err:", err)
	}
}

func TestRabbitMQ_SyncCallBackMsg(t *testing.T) {

	t.Parallel()
	mq := getMQ()
	defer mq.Close()
	//ch, err := mq.GetChannel()
	//if err != nil {
	//	t.Error("channel should be nil", err)
	//	return
	//}
	//defer ch.Close()
	go func() {
		fmt.Println("new declare wait--------")
		time.Sleep(3 * time.Second)
		mq.DeclareDirectEx("protocol.direct.sync")
		fmt.Println("new declare ok--------")
	}()
	fmt.Println("sync wait+++++++")
	msg, err := mq.SyncCallBackMsg("test", "", "test sync pub receive", false, 20)
	if err != nil {
		t.Error("sync callback msg err:", err)
		return
	}
	t.Log("sync callback msg:", string(msg))
	//_ = ch.Close()
	time.Sleep(10 * time.Second)
	t.Log("sync callback waiting finished")

}

func TestRabbitMQ_DeclareTmpQueue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	mq := getMQ()
	defer mq.Close()
	ch, _ := mq.GetChannel()
	defer ch.Close()
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(1000)

	tmpExchange := fmt.Sprintf("prootocol.tmp.e_%d_%d", time.Now().Unix(), num)
	tmpQueue := fmt.Sprintf("prootocol.tmp.q_%d_%d", time.Now().Unix(), num)
	fmt.Println("+++++++", tmpExchange, tmpQueue)
	tests := []struct {
		name  string
		e     string
		q     string
		isErr bool
	}{
		//{"全空", "", "", true},
		//{"队列空", tmpExchange, "", true},
		{"邮局空", "", tmpQueue, false},
		//{"全部非空", tmpExchange + "2", tmpQueue + "2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mq.DeclareTmpQueue(tt.e, tt.q)
			if tt.isErr != (err != nil) {
				t.Error("declare tmp queue err:", err)
			}
			time.Sleep(15 * time.Second)
			if tt.q != "" {
				_, err = ch.Consume(tt.q, "", false, false, false, false, nil)
				if tt.isErr != (err != nil) {
					t.Error("get msg err:", err)
				}
			}

			time.Sleep(6 * time.Second)
		})
	}

}

func TestRabbitMQ_Publish(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	mq := getMQ()
	defer mq.Close()

	err := mq.Publish("test", "", `{"name":"jack", "age":8}`)
	if err != nil {
		t.Error("publish err:", err)
	}
	time.Sleep(8 * time.Second)

	msg, err := mq.GetMsg("test.pub")
	if err != nil {
		t.Error("get msg err:", err)
	}
	u := &struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{}

	_ = json.Unmarshal([]byte(msg), u)
	fmt.Println(u)

}

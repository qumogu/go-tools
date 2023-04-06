package rabbitmq

import (
	"fmt"
	"net/url"
)

type RabbitMQConfig struct {
	Host          string
	Port          string
	Username      string
	Password      string
	VirtualHost   string
	PrefetchCount int
}

func (c *RabbitMQConfig) GetUri() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", c.Username, url.QueryEscape(c.Password), c.Host, c.Port, c.VirtualHost)
}

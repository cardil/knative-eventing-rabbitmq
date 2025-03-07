/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	Username string `envconfig:"USER" required:"true"`
	Password string `envconfig:"PASSWORD" required:"true"`
	Broker   string `envconfig:"RABBITBROKER" required:"true"`
	Count    int    `envconfig:"COUNT" default:"1"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatal("[ERROR] Failed to process env var: ", err)
	}
	connStr := fmt.Sprintf("amqp://%s:%s@%s", env.Username, env.Password, env.Broker)

	time.Sleep(1 * time.Minute)

	conn, err := amqp.Dial(connStr)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"logs",    // name
		"headers", // type
		true,      // durable
		false,     // auto-deleted
		false,     // internal
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare an exchange")
	var body, contentType string
	var headers amqp.Table

	for i := 0; i < env.Count; i++ {
		switch i % 3 {
		case 0:
			contentType = "application/json"
			headers = amqp.Table{
				"ce-specversion":     "1.0",
				"ce-id":              i,
				"ce-type":            "knative.producer.e2etest",
				"ce-source":          "example/source.uri",
				"ce-datacontenttype": "application/json; charset=UTF-8",
			}
			body = `{ "message": "Hello, BinCEWorld!" }`
		case 1:
			contentType = "application/cloudevents+json"
			headers = amqp.Table{}
			body = fmt.Sprintf(`{
				"id": "%d",
				"type": "knative.producer.e2etest",
				"source": "example/source.uri",
				"data": "Hello, CEWorld!",
				"specversion": "1.0",
				"datacontenttype": "text/plain"
			}`, i)
		case 2:
			contentType = "text/plain"
			headers = amqp.Table{}
			body = fmt.Sprintf(`{ "id": "%d", "message": "Hello, World!" }`, i)
		}

		err = ch.Publish(
			"logs", // exchange
			"",     // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: contentType,
				Body:        []byte(body),
				Headers:     headers,
			})
		failOnError(err, "Failed to publish a message")
		log.Printf(" [x] Sent %s", body)
		time.Sleep(50 * time.Millisecond)
	}
}

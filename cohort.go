package main

import (
	"bytes"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"utils"

	"github.com/streadway/amqp"
)

const TIMEOUT = 6 * time.Second

func replyToCoord(ch *amqp.Channel, msg string) (err error) {
	err = ch.Publish("", "coordinator_queue", false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	return
}

func main() {
	cohortCount := utils.GetCohortCount()
	log.Printf("Cohort count is %d\n", cohortCount)

	conn, err := amqp.Dial("amqp://guest:guest@coordinator:5672/")
	utils.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	utils.FailOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare("broadcast", "fanout", true, false, false, false, nil)
	utils.FailOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	utils.FailOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(q.Name, "", "broadcast", false, nil)
	utils.FailOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	utils.FailOnError(err, "Failed to register a consumer")

	var noAgree bool
	flag.BoolVar(&noAgree, "noAgree", false,
		"If true, the cohort will not agree to commit")
	var noAck bool
	flag.BoolVar(&noAck, "noAck", false,
		"If true, the cohort will crash before sending Ack")
	flag.Parse()

	state := utils.NotInitiated
	exit := false
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Can't get hostname!")
	}
	s := rand.NewSource(time.Now().UnixNano() + int64(hostname[len(hostname)-1]))
	rng := rand.New(s)
	log.Println("Awaiting CommitRequest")

	for !exit {
		var msg amqp.Delivery
		if state != utils.Aborted && state != utils.Committed {
			select {
			case msg = <-msgs:
				if bytes.Equal(msg.Body, []byte("Abort")) {
					log.Println("Received Abort, transitioning to Aborted state")
					time.Sleep(time.Second)
					state = utils.Aborted
				}
			case <-time.After(TIMEOUT):
				if state == utils.Prepared {
					log.Println("Timeout Occurred in Prepared state,",
						"transitioning to Committed state")
					time.Sleep(time.Second)
					state = utils.Committed
				} else {
					log.Println("Timeout occurred, transitioning to Aborted state")
					time.Sleep(time.Second)
					state = utils.Aborted
				}
			}
		}

		switch state {
		case utils.NotInitiated:
			if !bytes.Equal(msg.Body, []byte("CommitRequest")) {
				log.Println("Unexpected message received:", msg.Body)
				break
			}

			log.Println("Received CommitRequest")
			// <0.5; 4.5> seconds
			d := 500 * (rng.Intn(9) + 1)
			time.Sleep(time.Duration(d) * time.Millisecond)
			if noAgree {
				log.Println("Replying 'No' and transitioning to Aborted state")
				err := replyToCoord(ch, "No")
				utils.FailOnError(err, "Failed to reply")
				state = utils.Aborted
			} else {
				log.Println("Replying 'Yes' and transitioning to Waiting state")
				err := replyToCoord(ch, "Yes")
				utils.FailOnError(err, "Failed to reply")
				state = utils.Waiting
			}
		case utils.Waiting:
			if bytes.Equal(msg.Body, []byte("Prepare")) {
				log.Println("Received Prepare")
				d := 500 * (rng.Intn(9) + 1)
				time.Sleep(time.Duration(d) * time.Millisecond)
				if noAck {
					log.Println("CRASH!")
					exit = true
					break
				}
				log.Println("Sending Ack and transitioning to Prepared state")
				err := replyToCoord(ch, "Ack")
				utils.FailOnError(err, "Failed to send Ack to coordinator")
				state = utils.Prepared
			}
		case utils.Prepared:
			if bytes.Equal(msg.Body, []byte("Commit")) {
				log.Println("Received Commit, committing transaction",
					"and transitioning to Committed state")
				time.Sleep(time.Second)
				state = utils.Committed
			}
		case utils.Committed:
			log.Println("Transaction committed, exiting")
			time.Sleep(time.Second)
			exit = true
		case utils.Aborted:
			log.Println("Exiting")
			time.Sleep(time.Second)
			exit = true
		}
	}
}

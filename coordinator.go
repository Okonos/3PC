package main

import (
	"bytes"
	"flag"
	"log"
	"time"

	"utils"

	"github.com/streadway/amqp"
)

const TIMEOUT = 5 * time.Second

func broadcast(ch *amqp.Channel, msg string) (err error) {
	err = ch.Publish("broadcast", "", false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	return
}

func main() {
	cohortCount := utils.GetCohortCount()
	log.Printf("Cohort count is %d\n", cohortCount)

	log.Println("Initializing message broker connection")
	time.Sleep(time.Second)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	utils.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	utils.FailOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare("broadcast", "fanout", true, false, false, false, nil)
	utils.FailOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare("coordinator_queue", false, false, false, false, nil)
	utils.FailOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	utils.FailOnError(err, "Failed to register a consumer")

	var noCommit bool
	flag.BoolVar(&noCommit, "noCommit", false,
		"If true, the coordinator will crash after sending Prepare")
	var noPrepare bool
	flag.BoolVar(&noPrepare, "noPrepare", false,
		"If true, the coordinator will crash before sending Prepare")
	flag.Parse()

	// Init - broadcast Commit Request
	log.Println("Initiating commit - broadcasting Commit Request")
	err = broadcast(ch, "CommitRequest")
	utils.FailOnError(err, "Failed to broadcast CommitRequest")

	log.Println("Transitioning to Waiting state")
	state := utils.Waiting
	var msg amqp.Delivery
	replies := 0
	exit := false
	timeoutTimer := time.NewTimer(TIMEOUT)

	for !exit {
		if state != utils.Aborted && state != utils.Committed {
			select {
			case msg = <-msgs:
			case <-timeoutTimer.C:
				log.Println("Timeout occurred, transitioning to Aborted state")
				state = utils.Aborted
			}
		}

		switch state {
		case utils.Waiting:
			if bytes.Equal(msg.Body, []byte("No")) {
				log.Println("Cohort disagrees, transitioning to Aborted state")
				state = utils.Aborted
				break
			}
			replies++
			log.Println("Cohort agreed, number of cohorts agreeing so far:",
				replies)
			if replies == cohortCount {
				log.Println("All cohorts agreed, broadcasting Prepare",
					"and transitioning to Prepared state")
				if noPrepare {
					log.Println("CRASH!")
					exit = true
					break
				}
				err := broadcast(ch, "Prepare")
				utils.FailOnError(err, "Failed to broadcast Prepare")
				if noCommit {
					log.Println("CRASH!")
					exit = true
					break
				}
				replies = 0
				state = utils.Prepared
				resetTimer(timeoutTimer)
			}
		case utils.Prepared:
			if bytes.Equal(msg.Body, []byte("Ack")) {
				replies++
				log.Println("Received Ack, Ack count so far:", replies)
			}
			if replies == cohortCount {
				log.Println("Received Ack from all cohorts, broadcasting Commit",
					"and transitioning to Committed state")
				err := broadcast(ch, "Commit")
				utils.FailOnError(err, "Failed to broadcast Commit")
				state = utils.Committed
			}
		case utils.Committed:
			log.Println("Transaction successfully committed, exiting")
			time.Sleep(time.Second)
			exit = true
		case utils.Aborted:
			log.Println("Broadcasting Abort and exiting")
			err := broadcast(ch, "Abort")
			utils.FailOnError(err, "Failed to broadcast Abort")
			time.Sleep(time.Second)
			exit = true
		}
	}
}

func resetTimer(timer *time.Timer) {
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(TIMEOUT)
}

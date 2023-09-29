package main

import (
	"github.com/gorilla/mux"
)

const (
	dbString   = "admin:ammaryasser@tcp(universe.cbrsnlipsjis.eu-west-1.rds.amazonaws.com:3306)/testdb"
	queueName  = "chats"
	mqttString = "amqp://client-py:st@yhungry@ac7622565a1044e58a9e4a088efcd05d-190314016.eu-west-1.elb.amazonaws.com:5672/"
	redisURL   = "redis://a1885c2f187ac44ba9d66f258773d630-100103902.eu-west-1.elb.amazonaws.com:6379/1"
	osURL      = "https://search-staging-z3rrlu65yks6qbepqvweu5cm7q.eu-west-1.es.amazonaws.com"
	osUser     = "admin"
	osPass     = "Foob@r00"
)

func main() {
	as, cs, ms := initDependencies()

	router := mux.NewRouter()
	MakeHTTPTransport(as, cs, ms, router)
}

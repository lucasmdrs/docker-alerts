package main

import (
	"log"
	"os"
	"strings"

	"github.com/lucasmdrs/dockerstats"
)

var (
	filters []string
)

func main() {
	if containers, isSet := os.LookupEnv("CONTAINERS"); isSet {
		filters = strings.Split(containers, ",")
	}
	m := dockerstats.NewMonitor(dockerstats.DefaultCommunicator, filters...)
	for res := range m.Stream {
		if res.Error != nil {
			log.Fatal(res.Error.Error())
		}

		if len(res.Stats) == 0 {
			log.Println("No Docker containers running, output complete.")
			continue
		}

		for _, s := range res.Stats {
			evaluate(s)
		}
	}
}

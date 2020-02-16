package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lucasmdrs/dockerstats"
	"github.com/lucasmdrs/go-sendmail/mail"
	sendmailModels "github.com/lucasmdrs/go-sendmail/models"
	ccMap "github.com/orcaman/concurrent-map"
)

const (
	percentageSuffix = "%"
	alertTemplate    = `[ALERT]
	Container: %s
	%s`
)

var (
	destinations = strings.Split(os.Getenv("DESTINATIONS"), ",")
	hostname, _  = os.Hostname()

	gracePeriod              = time.Second * 60
	containerGracefulMapping = ccMap.New()

	memLimit float64 = 90
	cpuLimit float64 = 90
)

func init() {
	if hn, isSet := os.LookupEnv("HOSTNAME"); isSet {
		hostname = hn
	}

	if grace, isSet := os.LookupEnv("GRACE_PERIOD"); isSet {
		if sec, err := strconv.Atoi(grace); err == nil && sec > 60 {
			gracePeriod = time.Second * time.Duration(sec)
		}
	}

	if mem, isSet := os.LookupEnv("MEM_LIMIT"); isSet {
		if limit, err := strconv.ParseFloat(mem, 64); err == nil {
			memLimit = limit
		}
	}
	if cpu, isSet := os.LookupEnv("CPU_LIMIT"); isSet {
		if limit, err := strconv.ParseFloat(cpu, 64); err == nil {
			cpuLimit = limit
		}
	}
}

func startGracefulPeriod(key string) {
	<-time.After(gracePeriod)
	containerGracefulMapping.Remove(key)
}

func evaluate(stats dockerstats.Stats) {
	var (
		cpuMsg string
		memMsg string
	)
	parsedCPU, err := strconv.ParseFloat(strings.TrimSuffix(stats.CPU, percentageSuffix), 64)
	if err != nil {
		log.Fatalln(err.Error())
	}
	parsedMem, err := strconv.ParseFloat(strings.TrimSuffix(stats.Memory.Percent, percentageSuffix), 64)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, inCPUGrace := containerGracefulMapping.Get(stats.ContainerName + ".cpu")
	if parsedCPU > cpuLimit && !inCPUGrace {
		log.Println("NOTIFYING CPU!")
		cpuMsg = fmt.Sprintf("CPU limit reached: [Threshold: %.2f] [Value: %.2f]", cpuLimit, parsedCPU)
		notify(fmt.Sprintf(alertTemplate, stats.ContainerName, cpuMsg))
		containerGracefulMapping.Set(stats.ContainerName+".cpu", true)
		go startGracefulPeriod(stats.ContainerName + ".cpu")
	}
	_, inMemGrace := containerGracefulMapping.Get(stats.ContainerName + ".mem")
	if parsedMem > memLimit && !inMemGrace {
		log.Println("NOTIFYING MEMORY!")
		memMsg = fmt.Sprintf("Memory limit reached: [Threshold: %.2f] [Value: %.2f]", memLimit, parsedMem)
		notify(fmt.Sprintf(alertTemplate, stats.ContainerName, memMsg))
		containerGracefulMapping.Set(stats.ContainerName+".mem", true)
		go startGracefulPeriod(stats.ContainerName + ".mem")
	}
}

func notify(msg string) {
	parsedMessage := sendmailModels.MailInfo{
		FromMail:         "dockeralers@lucasmdrs.com",
		FromName:         "Docker Alerts",
		Subject:          hostname + " alert",
		PlainTextContent: msg,
		HTMLContent:      msg,
	}
	for _, to := range destinations {
		parsedMessage.To = to
		if _, err := mail.SendMail(parsedMessage); err != nil {
			log.Println(err.Error())
		}
	}
}

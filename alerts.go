package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/lucasmdrs/dockerstats"
	"github.com/lucasmdrs/go-sendmail/mail"
	sendmailModels "github.com/lucasmdrs/go-sendmail/models"
	ccMap "github.com/orcaman/concurrent-map"
)

const (
	percentageSuffix = "%"
	alertTemplate    = "template.html"
)

var (
	destinations = strings.Split(os.Getenv("DESTINATIONS"), ",")
	hostname, _  = os.Hostname()
	title        = "Docker Alerts"
	sender       = "docker@alerts.com"

	gracePeriod              = time.Second * 600
	containerGracefulMapping = ccMap.New()

	memLimit float64 = 90
	cpuLimit float64 = 90
)

type TemplateInfo struct {
	Value     string
	Threshold string
	Container string
	Hostname  string
	Message   string
}

func init() {
	if s, isSet := os.LookupEnv("EMAIL_SENDER"); isSet {
		sender = s
	}
	if t, isSet := os.LookupEnv("EMAIL_TITLE"); isSet {
		title = t
	}
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
		info := TemplateInfo{
			Container: stats.ContainerName,
			Hostname:  hostname,
			Message:   "CPU limit reached",
			Threshold: fmt.Sprintf("%.2f", cpuLimit),
			Value:     fmt.Sprintf("%.2f", parsedCPU),
		}
		notify(parseHTMLTemplate(info), parseTextTemplate(info))
		containerGracefulMapping.Set(stats.ContainerName+".cpu", true)
		go startGracefulPeriod(stats.ContainerName + ".cpu")
	}
	_, inMemGrace := containerGracefulMapping.Get(stats.ContainerName + ".mem")
	if parsedMem > memLimit && !inMemGrace {
		log.Println("NOTIFYING MEMORY!")
		info := TemplateInfo{
			Container: stats.ContainerName,
			Hostname:  hostname,
			Message:   "Memory limit reached",
			Threshold: fmt.Sprintf("%.2f", memLimit),
			Value:     fmt.Sprintf("%.2f", parsedMem),
		}
		notify(parseHTMLTemplate(info), parseTextTemplate(info))
		containerGracefulMapping.Set(stats.ContainerName+".mem", true)
		go startGracefulPeriod(stats.ContainerName + ".mem")
	}
}

func notify(html, text string) {
	if html == "" {
		html = text
	}
	parsedMessage := sendmailModels.MailInfo{
		FromMail:         sender,
		FromName:         title,
		Subject:          title,
		PlainTextContent: text,
		HTMLContent:      html,
	}
	for _, to := range destinations {
		parsedMessage.To = to
		if _, err := mail.SendMail(parsedMessage); err != nil {
			log.Println(err.Error())
		}
	}
}

func parseTextTemplate(data TemplateInfo) string {
	return fmt.Sprintf("[ %s - %s ] %s: [Threshold: %s] [Value: %s]",
		data.Hostname,
		data.Container,
		data.Message,
		data.Threshold,
		data.Value,
	)
}

func parseHTMLTemplate(data TemplateInfo) string {
	t := template.New("template.html")
	t, err := t.ParseFiles(alertTemplate)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		log.Println(err.Error())
		return ""
	}
	return tpl.String()
}

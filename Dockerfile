FROM golang as builder

WORKDIR /go/src/github.com/lucasmdrs/docker-alerts

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker-alerts . 
FROM docker:dind

WORKDIR /root/

ENV DESTINATIONS="example@example.com"
ENV SENDGRID_API_KEY="SG.KEY"
ENV CPU_LIMIT=90
ENV MEM_LIMIT=90
ENV HOSTNAME="undefined"

COPY --from=builder /go/src/github.com/lucasmdrs/docker-alerts/docker-alerts .

RUN chmod +x docker-alerts \
    && mv /usr/local/bin/docker /usr/bin/docker

ENTRYPOINT ["./docker-alerts"]

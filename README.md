# Docker Alerts

A process to run in backgroud, collecting containers stats and alerting it through Sendgrid email.

## [EXPERIMENTAL] This code has not been optimized or tested for a Production environment, use it at your own risk.

# Requirements
    - Docker
    - Sendgrid API Key

# Getting Started

Environments:
 - DESTINATIONS: A csv of destination emails to be sent (example@example.com,example2@example.com)
 - SENDGRID_API_KEY: The Sendgrid API Key (SG...)
 - CPU_LIMIT: The threshold value for a container CPU usage (90)
 - MEM_LIMIT: The threshold value for a container Memory usage (90)
 - GRACE_PERIOD: The time in seconds before sending the same alert for the container (Minimum 60)
 - HOSTNAME: An identifier for the current Host (My-Host)
 - CONTAINERS: A csv of container names or ids to filter by (dreamy_shirley,4ee48c5977b1)
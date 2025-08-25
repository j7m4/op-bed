#!/usr/bin/env bash

export OTEL_ENDPOINT="http://localhost:4318"

TIMESTAMP=$(date +%s%N)

curl "${OTEL_ENDPOINT}/v1/logs" \
-H "Content-Type: application/json" \
-d '{
  "resource_logs": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": { "string_value": "canary-curl-app" }
          }
        ]
      },
      "scope_logs": [
        {
          "scope": {},
          "log_records": [
            {
              "time_unix_nano": "'"${TIMESTAMP}"'",
              "observed_time_unix_nano": "'"${TIMESTAMP}"'",
              "severity_text": "INFO",
              "severity_number": 9,
              "body": { "string_value": "Canary log message [curl]" },
              "attributes": [
                {
                  "key": "user.id",
                  "value": { "string_value": "canary-curl" }
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}'
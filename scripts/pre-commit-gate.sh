#!/usr/bin/env bash

#Helper -> allow
allow() {
    printf '{"decision":"allow"}\n'
    exit 0
}

#Helper -> deny
deny() {
  jq -n --arg reason "$1" \
    '{"decision":"deny","reason":$reason}'
  exit 0
}

#read agy input
INPUT="$(cat)"

#check if the input contains git commit or merge
if ! printf '%s' "$INPUT" | grep -Eq 'git[[:space:]]+(commit|merge)'; then
    allow
fi

#Run go test
if ! (cd "../services/go-api" && go test ./...)>/dev/null 2>&1; then
    deny "Go tests failed. Fix Go tests before committing."
fi

#Run py test
if ! (cd "../services/py-ai" && .venv/bin/pytest)>/dev/null 2>&1; then
    deny "Python tests failed. Fix Python tests before committing."
fi

#Both tests passed
allow
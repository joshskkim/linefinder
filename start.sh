#!/bin/bash

# Load environment variables from .env file
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

if [ -z "$ODDS_API_KEY" ]; then
    echo "Error: ODDS_API_KEY is not set in .env file"
    exit 1
fi

echo "Starting LineFinder..."
go run ./cmd/server

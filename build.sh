#!/bin/bash

# Exit on error
set -e

# Build the backend
go build

# Build the React app
cd frontend
npm ci
npm run build:prod
cd ..

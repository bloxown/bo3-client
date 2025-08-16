#!/bin/bash
go build -o bo3client ./cmd/bo3client
go build -o bo3server ./cmd/bo3server
chmod +x ./bo3server
chmod +x ./bo3client
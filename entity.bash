#!/usr/bin/env bash
genna model -c postgres://postgres:12345@localhost:5432/blockchainextractor?sslmode=disable -o ./src/entity/entity.go -t public.* --pkg entity --gopg 9
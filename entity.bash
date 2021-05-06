#!/usr/bin/env bash
genna model -c postgres://postgres:12345@localhost:5432/cloverscan_contract_registry?sslmode=disable -o ./src/entity/entity.go -t public.* --pkg entity --gopg 9
#!/bin/bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/register -H "Content-Type: application/json" -d '{"email":"admin@example.test","password":"securepassword"}' | grep -oP '"access_token":"\K[^"]+')
if [ -z "$TOKEN" ]; then
  TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login -H "Content-Type: application/json" -d '{"email":"admin@example.test","password":"securepassword"}' | grep -oP '"access_token":"\K[^"]+')
fi
DOMAIN_ID=$(curl -s -X POST http://localhost:8080/v1/domains -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"example.test"}' | grep -oP '"id":"\K[^"]+')
curl -s -X POST http://localhost:8080/v1/domains/$DOMAIN_ID/addresses -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"local_part":"alice"}'
curl -s -X POST http://localhost:8080/v1/domains/$DOMAIN_ID/addresses -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"local_part":"bob"}'
sleep 6

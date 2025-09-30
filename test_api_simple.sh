#!/bin/bash

echo "Testing Admin APIs..."

echo "1. Testing Admin Apps API:"
curl -X GET "http://localhost:8080/api/v1/admin/apps?page=1&page_size=5" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s

echo -e "\n2. Testing Admin Groups API:"
curl -X GET "http://localhost:8080/api/v1/admin/groups?page=1&page_size=5" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s

echo -e "\n3. Testing Admin Users API:"
curl -X GET "http://localhost:8080/api/v1/admin/users?page=1&page_size=5" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  -s

echo -e "\nDone!"

#!/bin/bash
# Login and test build API
cd /c/Users/Master/Desktop/c2

echo "=== Login ==="
LOGIN_RESP=$(curl -sk -X POST https://localhost:9443/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"259a2daf10da"}')
echo "$LOGIN_RESP" | head -c 300
echo ""

TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
echo "Token: ${TOKEN:0:30}..."

echo ""
echo "=== Get Listeners ==="
LISTENERS=$(curl -sk https://localhost:9443/api/v1/listeners \
  -H "Authorization: Bearer $TOKEN")
echo "$LISTENERS" | head -c 300
echo ""

LISTENER_ID=$(echo "$LISTENERS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Listener ID: $LISTENER_ID"

echo ""
echo "=== Build Implant ==="
BUILD_RESP=$(curl -sk -X POST https://localhost:9443/api/v1/build \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"os\":\"windows\",\"arch\":\"amd64\",\"listener_id\":\"$LISTENER_ID\",\"connection_type\":\"reverse\"}")
echo "$BUILD_RESP"

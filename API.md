# City Stories Guide — API Endpoints & Testing Guide

## Setup

```bash
cd infra
docker compose up -d
```

Services:
| Service    | URL                        |
|------------|----------------------------|
| API        | http://localhost:8080       |
| MinIO Console | http://localhost:9001   |
| Prometheus | http://localhost:9090       |
| Grafana    | http://localhost:3000       |
| PostgreSQL | localhost:5432              |

Base URL: `http://localhost:8080`

---

## Health & Monitoring

### GET /healthz
Server liveness check.
```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

### GET /readyz
Readiness check with component statuses.
```bash
curl http://localhost:8080/readyz
# {"status":"ok","checks":[{"name":"server","status":"ok"},{"name":"database","status":"ok"}]}
```

### GET /metrics
Prometheus metrics.
```bash
curl http://localhost:8080/metrics
```

---

## Authentication (Rate limit: 5/min)

### POST /api/v1/auth/register
Register a new admin/user account.
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepass123",
    "name": "Admin User"
  }'
# Response: {"data":{user}, "tokens":{"access_token":"...","refresh_token":"...","expires_in":3600}}
```

### POST /api/v1/auth/login
Login with email/password.
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepass123"
  }'
```

### POST /api/v1/auth/device
Device-based auth (mobile app, no credentials needed).
```bash
curl -X POST http://localhost:8080/api/v1/auth/device \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "test-device-001",
    "language": "en"
  }'
```

### POST /api/v1/auth/refresh
Refresh an expired access token.
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'
```

### POST /api/v1/auth/google
Google Sign-In (requires GOOGLE_CLIENT_ID configured).
```bash
curl -X POST http://localhost:8080/api/v1/auth/google \
  -H "Content-Type: application/json" \
  -d '{"id_token": "GOOGLE_ID_TOKEN"}'
```

### POST /api/v1/auth/apple
Apple Sign-In (requires Apple credentials configured).
```bash
curl -X POST http://localhost:8080/api/v1/auth/apple \
  -H "Content-Type: application/json" \
  -d '{"code": "APPLE_AUTH_CODE"}'
```

---

## Convenience: Set Token Variable

After login/register, save the token for subsequent requests:
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@test.com","password":"testpass123","name":"Test Admin"}' \
  | jq -r '.tokens.access_token')

echo $TOKEN
```

---

## Public Endpoints (No Auth)

### GET /api/v1/cities
List active cities (cursor pagination).
```bash
curl "http://localhost:8080/api/v1/cities?limit=10"
# {"data":[...],"next_cursor":"...","has_more":false}
```

### GET /api/v1/cities/:id
Get a single city.
```bash
curl http://localhost:8080/api/v1/cities/1
```

### GET /api/v1/cities/:id/download-manifest
Get offline download manifest for a city.
```bash
curl "http://localhost:8080/api/v1/cities/1/download-manifest?language=en"
# {"data":[{story_id, audio_url, file_size_bytes}], "total_size_bytes":..., "total_stories":...}
```

### GET /api/v1/pois
List POIs for a city.
```bash
curl "http://localhost:8080/api/v1/pois?city_id=1&limit=10"
# Query params: city_id (required), status, type, cursor, limit, sort_by, sort_dir
```

### GET /api/v1/pois/:id
Get a single POI.
```bash
curl http://localhost:8080/api/v1/pois/1
```

### GET /api/v1/stories
List stories for a POI.
```bash
curl "http://localhost:8080/api/v1/stories?poi_id=1&language=en&limit=10"
# Query params: poi_id (required), language, status, cursor, limit, sort_by, sort_dir
```

### GET /api/v1/stories/:id
Get a single story.
```bash
curl http://localhost:8080/api/v1/stories/1
```

### GET /api/v1/nearby-stories (Rate limit: 10/min)
Find stories near a location.
```bash
curl "http://localhost:8080/api/v1/nearby-stories?lat=48.8566&lng=2.3522&radius=200&language=en"
# Optional: heading, speed, user_id
```

### GET /api/v1/listenings
List user's listening history.
```bash
curl "http://localhost:8080/api/v1/listenings?user_id=YOUR_USER_UUID&limit=10"
```

### POST /api/v1/listenings
Record a story listening event.
```bash
curl -X POST http://localhost:8080/api/v1/listenings \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "YOUR_USER_UUID",
    "story_id": 1,
    "completed": true,
    "lat": 48.8566,
    "lng": 2.3522
  }'
```

### POST /api/v1/reports
Report content.
```bash
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Content-Type: application/json" \
  -d '{
    "story_id": 1,
    "user_id": "YOUR_USER_UUID",
    "type": "wrong_fact",
    "comment": "The building was built in 1900, not 1800"
  }'
# type: wrong_location | wrong_fact | inappropriate_content
```

### POST /api/v1/device-tokens
Register device for push notifications.
```bash
curl -X POST http://localhost:8080/api/v1/device-tokens \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "YOUR_USER_UUID",
    "token": "fcm-device-token-here",
    "platform": "android"
  }'
```

### DELETE /api/v1/device-tokens
Unregister device.
```bash
curl -X DELETE http://localhost:8080/api/v1/device-tokens \
  -H "Content-Type: application/json" \
  -d '{"token": "fcm-device-token-here"}'
```

---

## Authenticated User Endpoints (Bearer Token Required)

### GET /api/v1/users/me
Get current user profile.
```bash
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/users/me
Schedule account for deletion (30-day grace period).
```bash
curl -X DELETE http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/users/me/restore
Cancel scheduled account deletion.
```bash
curl -X POST http://localhost:8080/api/v1/users/me/restore \
  -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/purchases/verify
Verify in-app purchase.
```bash
curl -X POST http://localhost:8080/api/v1/purchases/verify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "android",
    "transaction_id": "txn-123",
    "receipt": "receipt-data",
    "type": "city_pack",
    "city_id": 1,
    "price": 4.99
  }'
# type: city_pack | subscription | lifetime
```

### GET /api/v1/purchases/status
Get current purchase/subscription status.
```bash
curl http://localhost:8080/api/v1/purchases/status \
  -H "Authorization: Bearer $TOKEN"
```

---

## Admin Endpoints (Admin JWT Required)

> **Note:** To make a user an admin, update the database directly:
> ```bash
> docker exec -it csg-postgres psql -U citystories -d citystories \
>   -c "UPDATE users SET role = 'admin' WHERE email = 'admin@test.com';"
> ```
> Then re-login to get a new token with admin role.

### GET /api/v1/admin/stats
Dashboard statistics.
```bash
curl http://localhost:8080/api/v1/admin/stats \
  -H "Authorization: Bearer $TOKEN"
# Returns: total_cities, total_pois, total_stories, total_users, total_listenings, etc.
```

### Cities Management

**List all cities:**
```bash
curl "http://localhost:8080/api/v1/admin/cities?limit=20&include_deleted=false" \
  -H "Authorization: Bearer $TOKEN"
```

**Create city:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/cities \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paris",
    "name_ru": "Париж",
    "country": "France",
    "center_lat": 48.8566,
    "center_lng": 2.3522,
    "radius_km": 15.0,
    "is_active": true,
    "download_size_mb": 50.0
  }'
```

**Update city:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/cities/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paris",
    "country": "France",
    "center_lat": 48.8566,
    "center_lng": 2.3522,
    "radius_km": 20.0,
    "is_active": true,
    "download_size_mb": 55.0
  }'
```

**Delete city (soft-delete):**
```bash
curl -X DELETE http://localhost:8080/api/v1/admin/cities/1 \
  -H "Authorization: Bearer $TOKEN"
```

**Restore deleted city:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/cities/1/restore \
  -H "Authorization: Bearer $TOKEN"
```

### POIs Management

**List POIs:**
```bash
curl "http://localhost:8080/api/v1/admin/pois?city_id=1&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

**Create POI:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/pois \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "city_id": 1,
    "name": "Eiffel Tower",
    "name_ru": "Эйфелева башня",
    "lat": 48.8584,
    "lng": 2.2945,
    "type": "monument",
    "address": "Champ de Mars, 5 Avenue Anatole France",
    "interest_score": 95,
    "status": "active"
  }'
# type: building | street | park | monument | church | bridge | square | museum | district | other
```

**Update POI:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/pois/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "city_id": 1,
    "name": "Eiffel Tower",
    "lat": 48.8584,
    "lng": 2.2945,
    "type": "monument",
    "interest_score": 98,
    "status": "active"
  }'
```

**Delete POI:**
```bash
curl -X DELETE http://localhost:8080/api/v1/admin/pois/1 \
  -H "Authorization: Bearer $TOKEN"
```

### Stories Management

**List stories:**
```bash
curl "http://localhost:8080/api/v1/admin/stories?poi_id=1&language=en&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

**Create story:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/stories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "poi_id": 1,
    "language": "en",
    "text": "The Eiffel Tower was built in 1889 for the World Fair...",
    "layer_type": "general",
    "order_index": 1,
    "confidence": 90,
    "status": "active"
  }'
# layer_type: atmosphere | human_story | hidden_detail | time_shift | general
```

**Update story:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/stories/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "poi_id": 1,
    "language": "en",
    "text": "Updated story text...",
    "layer_type": "general",
    "confidence": 95,
    "status": "active"
  }'
```

**Delete story:**
```bash
curl -X DELETE http://localhost:8080/api/v1/admin/stories/1 \
  -H "Authorization: Bearer $TOKEN"
```

### Reports Management

**List reports:**
```bash
curl "http://localhost:8080/api/v1/admin/reports?status=new&limit=20" \
  -H "Authorization: Bearer $TOKEN"
# status: new | reviewed | resolved | dismissed
```

**Update report status:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/reports/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "reviewed"}'
```

**Disable story from report (moderation action):**
```bash
curl -X POST http://localhost:8080/api/v1/admin/reports/1/disable-story \
  -H "Authorization: Bearer $TOKEN"
```

**List reports for a POI:**
```bash
curl http://localhost:8080/api/v1/admin/pois/1/reports \
  -H "Authorization: Bearer $TOKEN"
```

### Inflation Jobs (AI Story Generation)

**Trigger story inflation for a POI:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/pois/1/inflate \
  -H "Authorization: Bearer $TOKEN"
# Max 3 inflation segments per POI
```

**List inflation jobs for a POI:**
```bash
curl http://localhost:8080/api/v1/admin/pois/1/inflation-jobs \
  -H "Authorization: Bearer $TOKEN"
```

### Audit Logs

**List audit trail:**
```bash
curl "http://localhost:8080/api/v1/admin/audit-logs?limit=20" \
  -H "Authorization: Bearer $TOKEN"
# Filters: actor_id, action, resource_type, status, created_from, created_to
# action: create | update | delete | restore | trigger | update_status | disable_story
# resource_type: city | poi | story | report | inflation_job
```

---

## Full End-to-End Test Script

```bash
#!/bin/bash
set -e
BASE=http://localhost:8080

echo "=== 1. Health checks ==="
curl -sf "$BASE/healthz" | jq .
curl -sf "$BASE/readyz" | jq .

echo "=== 2. Register user ==="
REGISTER=$(curl -sf -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"e2e@test.com","password":"testpass123","name":"E2E Tester"}')
echo "$REGISTER" | jq .
TOKEN=$(echo "$REGISTER" | jq -r '.tokens.access_token')
USER_ID=$(echo "$REGISTER" | jq -r '.data.id')

echo "=== 3. Promote to admin ==="
docker exec csg-postgres psql -U citystories -d citystories \
  -c "UPDATE users SET role = 'admin' WHERE email = 'e2e@test.com';"

echo "=== 4. Re-login as admin ==="
LOGIN=$(curl -sf -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"e2e@test.com","password":"testpass123"}')
TOKEN=$(echo "$LOGIN" | jq -r '.tokens.access_token')

echo "=== 5. Create city ==="
CITY=$(curl -sf -X POST "$BASE/api/v1/admin/cities" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test City","country":"Testland","center_lat":48.85,"center_lng":2.35,"radius_km":10,"is_active":true,"download_size_mb":1}')
echo "$CITY" | jq .
CITY_ID=$(echo "$CITY" | jq -r '.data.id')

echo "=== 6. Create POI ==="
POI=$(curl -sf -X POST "$BASE/api/v1/admin/pois" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"city_id\":$CITY_ID,\"name\":\"Test POI\",\"lat\":48.8584,\"lng\":2.2945,\"type\":\"monument\",\"interest_score\":80,\"status\":\"active\"}")
echo "$POI" | jq .
POI_ID=$(echo "$POI" | jq -r '.data.id')

echo "=== 7. Create story ==="
STORY=$(curl -sf -X POST "$BASE/api/v1/admin/stories" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"poi_id\":$POI_ID,\"language\":\"en\",\"text\":\"A test story about this place.\",\"layer_type\":\"general\",\"confidence\":90,\"status\":\"active\"}")
echo "$STORY" | jq .
STORY_ID=$(echo "$STORY" | jq -r '.data.id')

echo "=== 8. Public endpoints ==="
curl -sf "$BASE/api/v1/cities" | jq .
curl -sf "$BASE/api/v1/cities/$CITY_ID" | jq .
curl -sf "$BASE/api/v1/pois?city_id=$CITY_ID" | jq .
curl -sf "$BASE/api/v1/pois/$POI_ID" | jq .
curl -sf "$BASE/api/v1/stories?poi_id=$POI_ID" | jq .
curl -sf "$BASE/api/v1/stories/$STORY_ID" | jq .

echo "=== 9. Nearby stories ==="
curl -sf "$BASE/api/v1/nearby-stories?lat=48.8584&lng=2.2945&radius=500&language=en" | jq .

echo "=== 10. Record listening ==="
curl -sf -X POST "$BASE/api/v1/listenings" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"story_id\":$STORY_ID,\"completed\":true}" | jq .

echo "=== 11. User profile ==="
curl -sf "$BASE/api/v1/users/me" -H "Authorization: Bearer $TOKEN" | jq .

echo "=== 12. Admin stats ==="
curl -sf "$BASE/api/v1/admin/stats" -H "Authorization: Bearer $TOKEN" | jq .

echo "=== 13. Audit logs ==="
curl -sf "$BASE/api/v1/admin/audit-logs?limit=5" -H "Authorization: Bearer $TOKEN" | jq .

echo "=== 14. Download manifest ==="
curl -sf "$BASE/api/v1/cities/$CITY_ID/download-manifest?language=en" | jq .

echo ""
echo "All tests passed!"
```

---

## Error Response Format

All errors return:
```json
{
  "error": "description of the error",
  "trace_id": "unique-trace-id"
}
```

| Status | Meaning                              |
|--------|--------------------------------------|
| 200    | Success                              |
| 201    | Created                              |
| 400    | Validation error                     |
| 401    | Missing/invalid token                |
| 403    | Not admin (for admin endpoints)      |
| 404    | Resource not found                   |
| 409    | Conflict (duplicate email, etc.)     |
| 429    | Rate limit exceeded                  |
| 503    | Service unavailable                  |

## Pagination

All list endpoints use **cursor-based pagination**:
```
?cursor=OPAQUE_TOKEN&limit=20
```
Response includes:
```json
{
  "data": [...],
  "next_cursor": "...",
  "has_more": true
}
```
Max limit: 100. Default: 20.

## Validation Rules

| Field          | Rule                                   |
|----------------|----------------------------------------|
| email          | Valid format, max 254 chars            |
| password       | 8-72 bytes                             |
| coordinates    | lat: -90..90, lng: -180..180           |
| language       | ISO 639-1 (2 chars: en, ru, etc.)      |
| interest_score | 0-100                                  |
| radius         | 10-500m (nearby), 0.1-1000km (cities)  |
| poi type       | building, street, park, monument, church, bridge, square, museum, district, other |
| layer_type     | atmosphere, human_story, hidden_detail, time_shift, general |
| report type    | wrong_location, wrong_fact, inappropriate_content |

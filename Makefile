.PHONY: doctor generate-api check-generated-api check-generated-client-usage verify-planning verify verify-backend verify-admin verify-mobile verify-api audit demo-setup demo-reset demo-seed dev-up dev-down dev-logs mobile-start mobile-android mobile-ios mobile-test

# ── Developer sanity check ──────────────────────────────────────────

doctor:
	@bash scripts/doctor.sh

# ── API client generation ────────────────────────────────────────────

generate-api:
	bash backend/scripts/generate-clients.sh

verify-planning:
	@echo "==> Verifying planning artifacts…"
	python3 scripts/validate_tasks.py

# ── Package-level verification targets ───────────────────────────────

verify-backend:
	@echo "==> Verifying backend…"
	$(MAKE) -C backend lint
	$(MAKE) -C backend test
	$(MAKE) -C backend build

verify-admin:
	@echo "==> Verifying admin…"
	cd admin && npm run lint
	cd admin && npm run typecheck
	cd admin && npm run build

verify-mobile:
	@echo "==> Verifying mobile…"
	cd mobile && npm run lint
	cd mobile && npm run typecheck
	cd mobile && npm test

GENERATED_SCHEMAS = admin/src/api/generated/schema.ts mobile/src/api/generated/schema.ts

check-generated-api:
	@echo "==> Checking generated API schemas for drift…"
	$(MAKE) generate-api
	@if ! git diff --quiet -- $(GENERATED_SCHEMAS); then \
		echo ""; \
		echo "ERROR: Generated API schemas are out of date."; \
		echo "Run 'make generate-api' and commit the updated files:"; \
		echo ""; \
		git diff --stat -- $(GENERATED_SCHEMAS); \
		exit 1; \
	fi
	@echo "Generated API schemas are up to date."

check-generated-client-usage:
	@echo "==> Checking for raw apiClient usage in endpoint wrappers…"
	@bash scripts/check-generated-client-usage.sh

lint-api:
	@echo "==> Linting OpenAPI spec…"
	@bash scripts/lint-openapi.sh

verify-api: lint-api check-generated-api check-generated-client-usage

# ── Local demo data ──────────────────────────────────────────────────

demo-seed:
	$(MAKE) -C backend seed

demo-reset:
	$(MAKE) -C backend demo-reset

demo-setup:
	@echo "==> Starting full local stack (API + infrastructure)…"
	cd infra && docker compose up -d --build
	@echo "==> Waiting for API to be healthy…"
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do \
		wget -q --spider http://localhost:8080/healthz 2>/dev/null && break; \
		echo "  waiting…"; sleep 2; \
	done
	@echo "==> Seeding demo data…"
	$(MAKE) -C backend seed
	@echo ""
	@echo "Demo environment ready!"
	@echo "  Admin: admin@demo.local / demodemo"
	@echo "  User:  user@demo.local  / demodemo"
	@echo "  API:   http://localhost:8080"

# ── Local dev stack ───────────────────────────────────────────────────

dev-up:
	cd infra && docker compose up -d --build
	@echo ""
	@echo "Local stack is starting. Services:"
	@echo "  API:       http://localhost:8080"
	@echo "  MinIO:     http://localhost:9001 (minioadmin / minioadmin_secret)"
	@echo "  Grafana:   http://localhost:3000"
	@echo ""
	@echo "Check health: curl http://localhost:8080/healthz"

dev-down:
	cd infra && docker compose down

dev-logs:
	cd infra && docker compose logs -f api

# ── Mobile app ───────────────────────────────────────────────────────

mobile-start:
	cd mobile && npm ci && npx expo start

mobile-android:
	cd mobile && npm ci && npx expo start --android

mobile-ios:
	cd mobile && npm ci && npx expo run:ios

mobile-test:
	cd mobile && npm test

# ── Security audit ───────────────────────────────────────────────────

audit:
	@echo "==> Running Go vulnerability check…"
	cd backend && govulncheck ./...
	@echo "==> Running admin npm audit…"
	cd admin && npm audit --audit-level=high
	@echo "==> Running mobile npm audit…"
	cd mobile && npm audit --audit-level=high
	@echo ""
	@echo "All audits passed."

# ── Top-level verify (runs everything) ───────────────────────────────

verify: verify-planning verify-backend verify-admin verify-mobile verify-api
	@echo ""
	@echo "All verifications passed."

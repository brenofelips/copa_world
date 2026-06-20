.PHONY: tidy build up down logs ps

tidy:
	cd services/ingestion-api && go mod tidy
	cd services/score-service  && go mod tidy
	cd services/event-service  && go mod tidy
	cd services/web-server     && go mod tidy

build: tidy
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down -v

logs:
	docker-compose logs -f

ps:
	docker-compose ps

# Quick test: send a match-started event via Nginx (port 80)
test-ingest:
	curl -s -X POST http://localhost/events \
	  -H "Content-Type: application/json" \
	  -d '{"id":"EVT001","match":"BRA-MAR-13-06-2026","team_A":"Brazil","team_B":"Morocco","competition":{"title":"world-cup-2026","stage":"group_stage"},"event":"MATCH_STARTED","minute":0,"sequence":1,"payload":{}}' | jq .

test-goal:
	curl -s -X POST http://localhost/events \
	  -H "Content-Type: application/json" \
	  -d '{"id":"EVT002","match":"BRA-MAR-13-06-2026","team_A":"Brazil","team_B":"Morocco","competition":{"title":"world-cup-2026","stage":"group_stage"},"event":"GOAL","minute":27,"sequence":2,"payload":{"team_id":"MAR","player_id":"player_11","player_name":"Ismail Salmri"}}' | jq .

test-match:
	curl -s http://localhost:8080/matches/BRA-MAR-13-06-2026 | jq .

test-stream:
	curl -N http://localhost:8080/matches/BRA-MAR-13-06-2026/stream

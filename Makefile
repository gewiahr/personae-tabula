docker-test:
	docker-compose -f docker-compose.test.yml up -d

docker-test-down:
	docker-compose -f docker-compose.test.yml down

docker-prod:
	docker-compose -f docker-compose.prod.yml up -d

docker-prod-down:
	docker-compose -f docker-compose.prod.yml down
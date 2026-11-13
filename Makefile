.PHONY: setup start stop restart logs traffic clean garage-start

setup:
	@echo "Setting up nginx container environment..."
	@sudo mkdir -p /var/log/apt
	@sudo chmod 777 /var/log/apt
	@chmod +x generate_traffic.sh
	@echo "Setup complete!"

start:
	@echo "Starting nginx container..."
	@docker run -d --name datafox-nginx \
		-p 8080:80 \
		-v $(PWD)/nginx.conf:/etc/nginx/nginx.conf:ro \
		-v /var/log/apt:/var/log/nginx \
		nginx:alpine
	@echo "Nginx running at http://localhost:8080"
	@echo "Logs writing to /var/log/apt/"

stop:
	@echo "Stopping nginx container..."
	@docker stop datafox-nginx 2>/dev/null || true
	@docker rm datafox-nginx 2>/dev/null || true
	@echo "Container stopped"

restart: stop start

logs:
	@docker logs -f datafox-nginx

traffic:
	@./generate_traffic.sh

clean: stop
	@echo "Cleaning up logs..."
	@sudo rm -f /var/log/apt/*.log
	@echo "Cleanup complete"

status:
	@docker ps -f name=datafox-nginx

garage-start:
	@echo "Starting garage container..."
	@mkdir -p garage/data garage/meta
	@docker run -d --name datafox-garage \
		-p 3900:3900 -p 3901:3901 -p 3903:3903 \
		-v $(PWD)/garage/garage.toml:/etc/garage.toml:ro \
		-v $(PWD)/garage/data:/data \
		-v $(PWD)/garage/meta:/meta \
		dxflrs/garage:v1.0.1
	@echo "Waiting for garage to be ready..."
	@until docker exec datafox-garage /garage status >/dev/null 2>&1; do sleep 1; done
	@NODE_ID=$$(docker exec datafox-garage /garage node id -q | cut -d@ -f1); \
	docker exec datafox-garage /garage layout assign -z dc1 -c 1G $$NODE_ID; \
	docker exec datafox-garage /garage layout apply --version 1
	@docker exec datafox-garage /garage bucket create logs
	@echo "Garage running at http://localhost:3900, bucket 'logs' ready"

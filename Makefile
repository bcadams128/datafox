.PHONY: setup start stop restart logs traffic clean

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

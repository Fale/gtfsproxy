build:
	go build ./cmd/gtfsproxy
	env GOOS=linux GOARCH=arm64 go build -o ansible/roles/gtfsproxy/files/gtfsproxy	./cmd/gtfsproxy 

serve:
	go build ./cmd/gtfsproxy
	./gtfsproxy -vvv serve --high-ports

deploy: build
	cd ansible && ansible-playbook -i inventory main.yaml

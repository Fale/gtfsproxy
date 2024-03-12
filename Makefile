build:
	go build ./cmd/gtfsproxy
	env GOOS=linux GOARCH=arm64 go build -o ansible/roles/gtfsproxy/files/gtfsproxy	./cmd/gtfsproxy 

deploy: build
	cd ansible && ansible-playbook -i inventory main.yaml

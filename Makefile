.PHONY: build

NAMESPACE := xnaveira
PROJECT := alertdelay

build:
	docker build -t $(NAMESPACE)/$(PROJECT) .

push:
	docker push $(NAMESPACE)/$(PROJECT)

run:
	docker run -e SLACK_CHANNEL=$SLACK_CHANNEL -e RESROBOTCLIENT_API_KEY=$RESROBOTCLIENT_API_KEY -v $(shell pwd)/alertdelay.yaml:/app/alertdelay.yaml $(NAMESPACE)/$(PROJECT)
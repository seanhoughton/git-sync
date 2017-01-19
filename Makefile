NAME=seanhoughton/git-sync
TAG=latest

.PHONY: image

image:
	docker build -t $(NAME):$(TAG) .

default: image
NAME=seanhoughton/git-sync
TAG=latest

.PHONY: image

image:
	docker build -t $(NAME):$(TAG) .

test:
	docker run --rm --env-file=.env $(NAME):$(TAG)

debug:
	command docker run --rm -it --env-file=.env $(NAME):$(TAG) sh

default: image

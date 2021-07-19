include .env
export $(shell sed 's/=.*//' .env)

build:
	@rm -rf staticbaceknd && go build

start: build
	@./staticbackend -host localhost

deploy:
	CGO_ENABLED=0 go build
	scp staticbackend sb-poc:/home/dstpierre/sb

test:
	@JWT_SECRET=okdevmode go test --race --cover
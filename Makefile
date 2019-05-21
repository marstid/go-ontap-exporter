all: docker-build


docker-build:
	docker build -t go-netapp -f Dockerfile .

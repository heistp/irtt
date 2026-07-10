# Docker

It is possible to run IRTT using [Docker](https://docs.docker.com/install/).


## Client

The [irtt](irtt) wrapper script is provided to run IRTT using Docker with some
sane defaults. To use privileged options (i.e. using low ports) you might need
the adjust the script to run as `root`.


## Server

The IRTT server can be run using [Docker
Compose](https://docs.docker.com/compose/). The `docker-compose.yml` file can be
used as a starting point to run a public IRTT server.

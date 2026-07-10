FROM alpine:3.11 AS build

COPY . /opt/irtt

RUN apk --update add go git
RUN go get ./opt/irtt
RUN cd /opt/irtt && go install -v ./cmd/irtt


FROM alpine:3.11

COPY --from=build /root/go/bin/irtt /opt

ENTRYPOINT ["/opt/irtt"]
CMD []

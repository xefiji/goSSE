FROM golang:1.13

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN go get -u github.com/gorilla/mux
RUN go get -u github.com/dgrijalva/jwt-go

# source env var ?

CMD ["app"]
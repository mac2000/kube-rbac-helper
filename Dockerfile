FROM golang:1.17-alpine as build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /kube-rbac-helper

FROM alpine


WORKDIR /

COPY --from=build /kube-rbac-helper /kube-rbac-helper

EXPOSE 8080

ENTRYPOINT ["/kube-rbac-helper"]

# docker build -t mac2000/kube-rbac-helper .
# docker run -it --rm --name=kube-rbac-helper -p 8080:8080 -v /Users/mac/Documents/dotfiles/kube/cub.yml:/cub.yml -e KUBECONFIG=/cub.yml mac2000/kube-rbac-helper

# docker push mac2000/kube-rbac-helper


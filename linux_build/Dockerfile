FROM golang:1.9.0-stretch

ENV PROJECT_PATH /app
ENV PATH $PROJECT_PATH/bin:$PATH

RUN mkdir -p $PROJECT_PATH && chmod -R 755 "$PROJECT_PATH"
WORKDIR $PROJECT_PATH

RUN go get github.com/constabulary/gb/... && \
  go install github.com/constabulary/gb

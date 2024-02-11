# interative dockerfile, TODO: build app + github action
FROM debian
RUN apt-get update && apt-get install -y ca-certificates
COPY ./koncord /koncord
ENTRYPOINT /koncord

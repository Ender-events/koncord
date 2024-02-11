# interative dockerfile, TODO: build app + github action
FROM debian
COPY ./koncord /koncord
ENTRYPOINT /koncord

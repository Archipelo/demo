FROM anchore/syft AS syft
WORKDIR /

FROM alpine
RUN apk add libc6-compat curl

ENV PATH "$PATH:/go/bin"
COPY go/bin/sbom /go/bin/

COPY --from=syft /syft /usr/bin/
COPY docker/config/syft/.syft.yaml /

ADD https://github.com/interlynk-io/sbomqs/releases/download/v0.0.19/sbomqs-linux-amd64 /go/bin/sbomqs
RUN chmod +rx /go/bin/sbomqs

RUN adduser --shell /bin/bash --system test
USER test
ENTRYPOINT /go/bin/sbom
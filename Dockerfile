
FROM alpine:latest

COPY ./comon /comon/comon
RUN chmod a+x /comon/comon

EXPOSE 9999

WORKDIR /gogetter
CMD ["/comon/comon"]

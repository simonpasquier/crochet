FROM busybox

COPY ./crochet /crochet

EXPOSE 8080
ENTRYPOINT [ "/crochet" ]
CMD []

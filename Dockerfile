FROM busybox

COPY ./webhook_ui /webhook_ui

EXPOSE 8080
ENTRYPOINT [ "/webhook_ui" ]
CMD []
